package errors

import "errors"

var FailedDownloadError = errors.New("kpm: failed to download dependency")
var CheckSumMismatchError = errors.New("kpm: checksum mismatch")
var FailedToVendorDependency = errors.New("kpm: failed to vendor dependency")
var FailedToPackage = errors.New("kpm: failed to package.")
var InvalidDependency = errors.New("kpm: invalid dependency.")
var InternalBug = errors.New("kpm: internal bug, please contact us and we will fix the problem.")
var FailedToLoadPackage = errors.New("kpm: failed to load package, please check the package path is valid.")

// Invalid Options Format Errors
// Invalid 'kpm init'
var InvalidInitOptions = errors.New("kpm: invalid 'kpm init' argument, you must provide a name for the package to be initialized.")

// Invalid 'kpm add'
var InvalidAddOptionsWithoutRegistry = errors.New("kpm: invalid 'kpm add' argument, you must provide a registry url for the package.")
var InvalidAddOptionsInvalidGitUrl = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Url for the package.")
var InvalidAddOptionsInvalidGitTag = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Tag for the package.")

// Invalid 'kpm run'
var InvalidRunOptionsWithoutEntryFiles = errors.New("kpm: invalid 'kpm run' argument, you must provide an entry file.")
var EntryFileNotFound = errors.New("kpm: entry file cannot be found, please make sure the '--input' entry file can be found")
var CompileFailed = errors.New("kpm: failed to compile kcl, please check the command 'kclvm_cli run' is still works.")
var FailedUnTarKclPackage = errors.New("kpm: failed to untar kcl package, please re-download")
var UnknownTarFormat = errors.New("kpm: unknown tar format.")
var InvalidKclPacakgeTar = errors.New("kpm: the '--tar' path is an invalid *.tar file")

// Invalid KPM_HOME
var InvalidKpmHomeInCurrentPkg = errors.New("kpm: environment variable KPM_HOME cannot be set to the same path as the current KCL package.")

// Invalid oci
var FailedLogin = errors.New("kpm: failed to login, please check registry, username and password is valid.")
var FailedLogout = errors.New("kpm: failed to logout, the registry not logged in.")
