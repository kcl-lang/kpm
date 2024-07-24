package errors

import (
	"fmt"
)

// KpmError is a custom error type for kpm
type KpmError struct {
	Message string
	Err     error
	Context string
}

func (e *KpmError) Error() string {
	return fmt.Sprintf("%s: %s - %v", e.Message, e.Context, e.Err)
}

func NewKpmError(message string, err error, context string) error {
	return &KpmError{
		Message: message,
		Err:     err,
		Context: context,
	}
}

// Error variables
var (
	FailedDownloadError           = NewKpmError("Operation failed", fmt.Errorf("failed to download dependency"), "")
	CheckSumMismatchError         = NewKpmError("Operation failed", fmt.Errorf("checksum mismatch"), "")
	FailedToVendorDependency      = NewKpmError("Operation failed", fmt.Errorf("failed to vendor dependency"), "")
	FailedToPackage               = NewKpmError("Operation failed", fmt.Errorf("failed to package"), "")
	InvalidDependency             = NewKpmError("Operation failed", fmt.Errorf("invalid dependency"), "")
	InternalBug                   = NewKpmError("Operation failed", fmt.Errorf("internal bug, please contact us and we will fix the problem"), "")
	FailedToLoadPackage           = NewKpmError("Operation failed", fmt.Errorf("failed to load package, please check the package path is valid"), "")
	InvalidInitOptions            = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm init' argument, you must provide a name for the package to be initialized"), "")
	InvalidAddOptions             = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm add' argument, you must provide a package name or url for the package"), "")
	InvalidAddOptionsInvalidGitUrl = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm add' argument, you must provide a Git Url for the package"), "")
	InvalidAddOptionsInvalidOciRef = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm add' argument, you must provide a valid Oci Ref for the package"), "")
	InvalidAddOptionsInvalidOciReg = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm add' argument, you must provide a Reg for the package"), "")
	InvalidAddOptionsInvalidOciRepo = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm add' argument, you must provide a Repo for the package"), "")
	MultipleSources               = NewKpmError("Operation failed", fmt.Errorf("multiple sources found, there must be a single source"), "")
	InvalidRunOptionsWithoutEntryFiles = NewKpmError("Operation failed", fmt.Errorf("invalid 'kpm run' argument, you must provide an entry file"), "")
	EntryFileNotFound             = NewKpmError("Operation failed", fmt.Errorf("entry file cannot be found, please make sure the '--input' entry file can be found"), "")
	CompileFailed                 = NewKpmError("Operation failed", fmt.Errorf("failed to compile kcl"), "")
	FailedUnTarKclPackage         = NewKpmError("Operation failed", fmt.Errorf("failed to untar kcl package, please re-download"), "")
	UnknownTarFormat              = NewKpmError("Operation failed", fmt.Errorf("unknown tar format"), "")
	KclPacakgeTarNotFound         = NewKpmError("Operation failed", fmt.Errorf("the kcl package tar path is not found"), "")
	InvalidKclPacakgeTar          = NewKpmError("Operation failed", fmt.Errorf("the kcl package tar path is an invalid *.tar file"), "")
	InvalidKpmHomeInCurrentPkg    = NewKpmError("Operation failed", fmt.Errorf("environment variable KCL_PKG_PATH cannot be set to the same path as the current KCL package"), "")
	FailedLogin                   = NewKpmError("Operation failed", fmt.Errorf("failed to login, please check registry, username and password is valid"), "")
	FailedLogout                  = NewKpmError("Operation failed", fmt.Errorf("failed to logout, the registry not logged in"), "")
	FailedPull                    = NewKpmError("Operation failed", fmt.Errorf("failed to pull kcl package"), "")
	FailedPushToOci               = NewKpmError("Operation failed", fmt.Errorf("failed to push kcl package tar to oci"), "")
	InvalidOciRef                 = NewKpmError("Operation failed", fmt.Errorf("invalid oci reference"), "")
	NotOciUrl                     = NewKpmError("Operation failed", fmt.Errorf("url is not an oci url"), "")
	IsOciRef                      = NewKpmError("Operation failed", fmt.Errorf("oci ref is not an url"), "")
	InvalidVersionFormat          = NewKpmError("Operation failed", fmt.Errorf("failed to parse version"), "")
	PathNotFound                  = NewKpmError("Operation failed", fmt.Errorf("path not found"), "")
	PathIsEmpty                   = NewKpmError("Operation failed", fmt.Errorf("path is empty"), "")
	InvalidPkg                    = NewKpmError("Operation failed", fmt.Errorf("invalid kcl package"), "")
	InvalidOciUrl                 = NewKpmError("Operation failed", fmt.Errorf("invalid oci url"), "")
	UnknownEnv                    = NewKpmError("Operation failed", fmt.Errorf("invalid environment variable"), "")
	NoKclFiles                    = NewKpmError("Operation failed", fmt.Errorf("No input KCL files"), "")
)
