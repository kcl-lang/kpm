// Copyright 2022 The KCL Authors. All rights reserved.

package reporter

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Init the log.
func InitReporter() {
	InitReporterWithLevel(LogLevelInfo)
}

// InitReporterWithLevel initializes the reporter with the given log level.
//
// At LogLevelInfo (default), only user-facing messages are emitted. The
// underlying errors attached to KpmEvent are kept internal.
//
// At LogLevelDebug, every detail is emitted, including the underlying error
// chain attached to KpmEvent.
//
// At LogLevelError and below, only fatal errors are emitted.
func InitReporterWithLevel(level LogLevel) {
	log.SetFlags(0)
	logrus.SetLevel(logrus.ErrorLevel)
	currentLogLevel = level
	switch level {
	case LogLevelDebug:
		log.SetPrefix("[debug] ")
	case LogLevelError:
		log.SetPrefix("[error] ")
	default:
		log.SetPrefix("[info] ")
	}
}

// LogLevel represents the verbosity of kpm output.
type LogLevel int

const (
	// LogLevelError suppresses everything except fatal errors.
	LogLevelError LogLevel = iota
	// LogLevelInfo is the default verbosity: user-facing messages only.
	LogLevelInfo
	// LogLevelDebug emits every detail, including underlying error chains.
	LogLevelDebug
)

// currentLogLevel is the active log level. It is set by InitReporterWithLevel
// and read by Fatal/LogDebug so output can be filtered at the source.
var currentLogLevel LogLevel = LogLevelInfo

// ParseLogLevel parses a level string (case-insensitive). Empty or unknown
// values resolve to LogLevelInfo to preserve the previous default behaviour.
func ParseLogLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LogLevelDebug
	case "error", "err":
		return LogLevelError
	case "warn", "warning":
		// Treated as info — kpm does not yet distinguish a separate warn tier,
		// but accepting "warn" here avoids silent mis-parsing for users who
		// set KPM_LOG_LEVEL based on other tools' conventions.
		return LogLevelInfo
	case "", "info":
		return LogLevelInfo
	default:
		return LogLevelInfo
	}
}

// GetLogLevel returns the currently configured log level.
func GetLogLevel() LogLevel {
	return currentLogLevel
}

// LogDebug prints a debug-level message. It is a no-op at LogLevelInfo or
// LogLevelError so it is safe to call from hot paths.
func LogDebug(v ...any) {
	if currentLogLevel >= LogLevelDebug {
		log.Println(v...)
	}
}

// Report prints to the logger.
// Arguments are handled in the manner of fmt.Println.
func Report(v ...any) {
	log.Println(v...)
}

// ExitWithReport prints to the logger and exit with 0.
// Arguments are handled in the manner of fmt.Println.
func ExitWithReport(v ...any) {
	log.Println(v...)
	os.Exit(0)
}

// Fatal prints to the logger and exit with 1.
// Arguments are handled in the manner of fmt.Println.
//
// When the active level is below LogLevelDebug and the only argument is a
// *KpmEvent with a populated msg, only the user-facing msg is printed — the
// underlying error attached to the event is suppressed to keep output tidy.
func Fatal(v ...any) {
	if msg, ok := fatalFilteredMessage(currentLogLevel, v); ok {
		log.Print(msg)
		os.Exit(1)
		return
	}
	log.Fatal(v...)
}

// fatalFilteredMessage applies the level-based filter used by Fatal. It
// returns the message that should be printed and true when the filter
// engaged; otherwise it returns ("", false) and Fatal falls back to the
// standard log.Fatal path.
func fatalFilteredMessage(level LogLevel, args []any) (string, bool) {
	if level >= LogLevelDebug || len(args) != 1 {
		return "", false
	}
	e, ok := args[0].(*KpmEvent)
	if !ok {
		return "", false
	}
	msg := strings.TrimSpace(e.Event())
	if msg == "" {
		return "", false
	}
	return msg, true
}

// Event is the interface that specifies the event used to show logs to users.
type Event interface {
	Event() string
}

type EventType int

const (
	Default EventType = iota

	// errors event type means the event is an error.
	InvalidRepo
	FailedNewOciClient
	RepoNotFound
	FailedLoadSettings
	FailedLoadCredential
	FailedCreateOciClient
	FailedSelectLatestVersion
	FailedSelectLatestCompatibleVersion
	FailedGetReleases
	FailedTopologicalSort
	FailedGetVertexProperties
	FailedGenerateSource
	FailedGetPackageVersions
	FailedCreateStorePath
	FailedPush
	FailedGetPkg
	FailedVendor
	FailedAccessPkgPath
	UnKnownPullWhat
	UnknownEnv
	InvalidKclPkg
	FailedUntarKclPkg
	FailedLoadKclMod
	FailedLoadKclModLock
	FailedCreateFile
	FailedPackage
	FailedLogin
	FailedLogout
	FileExists
	CheckSumMismatch
	CalSumFailed
	InvalidKpmHomeInCurrentPkg
	InvalidCmd
	InvalidPkgRef
	InvalidGitUrl
	WithoutGitTag
	FailedCloneFromGit
	FailedHashPkg
	FailedUpdatingBuildList
	Bug

	// normal event type means the event is a normal event.
	PullingStarted
	PullingFinished
	Pulling
	InvalidFlag
	Adding
	WaitingLock
	IsNotUrl
	IsNotRef
	UrlSchemeNotOci
	UnsupportOciUrlScheme
	SelectLatestVersion
	DownloadingFromOCI
	DownloadingFromGit
	LocalPathNotExist
	PathIsEmpty
	DependencyNotFoundInOrderedMap
	DependencyNotSetInOrderedMap
	ConflictPkgName
	AddItselfAsDep
	PkgTagExists
	DependencyNotFound
	CircularDependencyExist
	RemoveDep
	AddDep
	KclModNotFound
	CompileFailed
	FailedParseVersion
	FailedFetchOciManifest
)

// KpmEvent is the event used to show kpm logs to users.
type KpmEvent struct {
	errType EventType
	msg     string
	err     error
}

// Type returns the event type.
func (e *KpmEvent) Type() EventType {
	return e.errType
}

// Error makes KpmEvent can be used as an error.
func (e *KpmEvent) Error() string {
	result := ""
	if e.msg != "" {
		// append msg
		result = fmt.Sprintf("%s\n", e.msg)
	}
	if e.err != nil {
		result = fmt.Sprintf("%s%s\n", result, e.err.Error())
	}
	return result
}

// Event returns the msg of the event without error message.
func (e *KpmEvent) Event() string {
	if e.msg != "" {
		return fmt.Sprintf("%s\n", e.msg)
	}
	return ""
}

// NewErrorEvent returns a new KpmEvent with error.
func NewErrorEvent(errType EventType, err error, args ...string) *KpmEvent {
	return &KpmEvent{
		errType: errType,
		msg:     strings.Join(args, ""),
		err:     err,
	}
}

// NewEvent returns a new KpmEvent without error.
func NewEvent(errType EventType, args ...string) *KpmEvent {
	return &KpmEvent{
		errType: errType,
		msg:     strings.Join(args, ""),
		err:     nil,
	}
}

// ReportEventToStdout reports the event to users to stdout.
func ReportEventToStdout(event *KpmEvent) {
	fmt.Fprintf(os.Stdout, "%v", event.Event())
}

// ReportEventToStderr reports the event to users to stderr.
func ReportEventToStderr(event *KpmEvent) {
	fmt.Fprintf(os.Stderr, "%v", event.Event())
}

// ReportEvent reports the event to users to stdout.
func ReportEventTo(event *KpmEvent, w io.Writer) {
	if w != nil {
		fmt.Fprintf(w, "%v", event.Event())
	}
}

func ReportMsgTo(msg string, w io.Writer) {
	if w != nil {
		fmt.Fprintf(w, "%s\n", msg)
	}
}
