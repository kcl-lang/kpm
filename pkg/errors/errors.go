package errors

import "errors"

var FailedDownloadError = errors.New("kpm: failed to download dependency")
var CheckSumMismatchError = errors.New("kpm: checksum mismatch")
var FailedToVendorDependency = errors.New("kpm: failed to vendor dependency")
var FailedToPackage = errors.New("kpm: failed to package.")
var InvalidDependency = errors.New("kpm: invalid dependency.")
var InternalBug = errors.New("kpm: internal bug, please contact us and we will fix the problem.")

// Invalid Options Format Errors
// Invalid 'kpm init'
var InvalidInitOptions = errors.New("kpm: invalid 'kpm init' argument, you must provide a name for the package to be initialized.")

// Invalid 'kpm add'
var InvalidAddOptionsWithoutRegistry = errors.New("kpm: invalid 'kpm add' argument, you must provide a registry url for the package.")
var InvalidAddOptionsInvalidGitUrl = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Url for the package.")
var InvalidAddOptionsInvalidGitTag = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Tag for the package.")

// Invalid 'kpm run'
var InvalidRunOptionsWithoutEntryFiles = errors.New("kpm: invalid 'kpm run' argument, you must provide an entry file.")
var CompileFailed = errors.New("kpm: failed to compile kcl, please check the command 'kclvm_cli run' is still works.")

// invalid KPM_HOME
var InvalidKpmHomeInCurrentPkg = errors.New("kpm: environment variable KPM_HOME cannot be set to the same path as the current KCL package.")
