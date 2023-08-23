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
	log.SetFlags(0)
	logrus.SetLevel(logrus.ErrorLevel)
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
func Fatal(v ...any) {
	log.Fatal(v...)
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
	FailedGetPackageVersions
	FailedCreateStorePath
	FailedPush
	FailedGetPkg
	FailedAccessPkgPath
	UnKnownPullWhat
	UnknownEnv
	InvalidKclPkg
	FailedUntarKclPkg
	FailedLoadKclMod
	CheckSumMismatch
	CalSumFailed
	InvalidKpmHomeInCurrentPkg
	InvalidCmd
	InvalidPkgRef
	InvalidGitUrl
	WithoutGitTag
	FailedCloneFromGit
	FailedHashPkg
	Bug

	// normal event type means the event is a normal event.
	PullingStarted
	PullingFinished
	Pulling
	Adding
	IsNotUrl
	IsNotRef
	UrlSchemeNotOci
	UnsupportOciUrlScheme
	SelectLatestVersion
	DownloadingFromOCI
	DownloadingFromGit
	LocalPathNotExist
	PathIsEmpty
	AddItselfAsDep
	PkgTagExists
	DependencyNotFound
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
		result = fmt.Sprintf("kpm: %s\n", e.msg)
	}
	if e.err != nil {
		// append err
		_, ok := e.err.(*KpmEvent)
		if !ok {
			result = fmt.Sprintf("%skpm: %s\n", result, e.err.Error())
		} else {
			result = fmt.Sprintf("%s%s\n", result, e.err.Error())
		}
	}
	return result
}

// Event returns the msg of the event without error message.
func (e *KpmEvent) Event() string {
	if e.msg != "" {
		return fmt.Sprintf("kpm: %s\n", e.msg)
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
	fmt.Fprintf(w, "%v", event.Event())
}
