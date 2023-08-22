package errors

import (
	"errors"
)

var FailedDownloadError = errors.New("kpm: failed to download dependency")
var CheckSumMismatchError = errors.New("checksum mismatch")
var FailedToVendorDependency = errors.New("kpm: failed to vendor dependency")
var FailedToPackage = errors.New("kpm: failed to package.")
var InvalidDependency = errors.New("kpm: invalid dependency.")
var InternalBug = errors.New("kpm: internal bug, please contact us and we will fix the problem.")
var FailedToLoadPackage = errors.New("kpm: failed to load package, please check the package path is valid.")

// Invalid Options Format Errors
// Invalid 'kpm init'
var InvalidInitOptions = errors.New("kpm: invalid 'kpm init' argument, you must provide a name for the package to be initialized.")

// Invalid 'kpm add'
var InvalidAddOptions = errors.New("invalid 'kpm add' argument, you must provide a package name or url for the package.")
var InvalidAddOptionsInvalidGitUrl = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Url for the package.")
var InvalidAddOptionsInvalidOciRef = errors.New("kpm: invalid 'kpm add' argument, you must provide a valid Oci Ref for the package.")

var InvalidAddOptionsInvalidOciReg = errors.New("invalid 'kpm add' argument, you must provide a Reg for the package.")
var InvalidAddOptionsInvalidOciRepo = errors.New("invalid 'kpm add' argument, you must provide a Repo for the package.")

// Invalid 'kpm run'
var InvalidRunOptionsWithoutEntryFiles = errors.New("kpm: invalid 'kpm run' argument, you must provide an entry file.")
var EntryFileNotFound = errors.New("kpm: entry file cannot be found, please make sure the '--input' entry file can be found")
var CompileFailed = errors.New("kpm: failed to compile kcl.")
var FailedUnTarKclPackage = errors.New("kpm: failed to untar kcl package, please re-download")
var UnknownTarFormat = errors.New("kpm: unknown tar format.")
var KclPacakgeTarNotFound = errors.New("kpm: the kcl package tar path is not found")
var InvalidKclPacakgeTar = errors.New("kpm: the kcl package tar path is an invalid *.tar file")

// Invalid KCL_PKG_PATH
var InvalidKpmHomeInCurrentPkg = errors.New("environment variable KCL_PKG_PATH cannot be set to the same path as the current KCL package.")

// Invalid oci
var FailedLogin = errors.New("kpm: failed to login, please check registry, username and password is valid.")
var FailedLogout = errors.New("kpm: failed to logout, the registry not logged in.")
var FailedPull = errors.New("failed to pull kcl package.")
var FailedPushToOci = errors.New("kpm: failed to push kcl package tar to oci.")
var InvalidOciRef = errors.New("invalid oci reference.")
var NotOciUrl = errors.New("kpm: url is not an oci url.")
var IsOciRef = errors.New("kpm: oci ref is not an url.")

// Invalid Version
var InvalidVersionFormat = errors.New("kpm: failed to parse version.")
var PathNotFound = errors.New("path not found.")
var PathIsEmpty = errors.New("path is empty.")
var InvalidPkg = errors.New("invalid kcl package.")
var InvalidOciUrl = errors.New("invalid oci url.")
var UnknownEnv = errors.New("invalid environment variable.")

// No kcl files
var NoKclFiles = errors.New("No input KCL files")
