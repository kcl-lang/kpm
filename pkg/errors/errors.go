package errors

import "errors"

var FailedDownloadError = errors.New("kpm: failed to download dependency")
var CheckSumMismatchError = errors.New("kpm: checksum mismatch")
var FailedToVendorDependency = errors.New("kpm: failed to vendor dependency")
var FailedToPackage = errors.New("kpm: failed to package.")
var InvalidDependency = errors.New("kpm: invalid dependency.")
var InternalBug = errors.New("kpm: internal bug.")

// Invalid Options Format Errors
// Invalid 'kpm init'
var InvalidInitOptions = errors.New("kpm: invalid 'kpm init' argument, you must provide a name for the package to be initialized.")

// Invalid 'kpm add'
var InvalidAddOptionsWithoutRegistry = errors.New("kpm: invalid 'kpm add' argument, you must provide a registry url for the package.")
var InvalidAddOptionsInvalidGitUrl = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Url for the package.")
var InvalidAddOptionsInvalidGitTag = errors.New("kpm: invalid 'kpm add' argument, you must provide a Git Tag for the package.")
