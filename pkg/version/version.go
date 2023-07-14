// Copyright 2022 The KCL Authors. All rights reserved.

package version

// version will be set by build flags.
var version string

// GetVersionInStr() will return the latest version of kpm.
func GetVersionInStr() string {
	if len(version) == 0 {
		// If version is not set by build flags, return the version constant.
		return KpmAbiVersion.String()
	}
	return version
}

// KpmVersionType is the version type of kpm.
type KpmVersionType string

// String() will transform KpmVersionType to string.
func (kvt KpmVersionType) String() string {
	return string(kvt)
}

// All the kpm versions.
const (
	KpmAbiVersion         KpmVersionType = KpmAbiVersion_0_3_0
	KpmVersionType_latest                = KpmAbiVersion_0_3_0

	KpmAbiVersion_0_3_0 KpmVersionType = "0.3.0"
	KpmAbiVersion_0_2_6 KpmVersionType = "0.2.6"
	KpmAbiVersion_0_2_5 KpmVersionType = "0.2.5"
	KpmAbiVersion_0_2_4 KpmVersionType = "0.2.4"
	KpmAbiVersion_0_2_3 KpmVersionType = "0.2.3"
	KpmAbiVersion_0_2_2 KpmVersionType = "0.2.2"
	KpmAbiVersion_0_2_1 KpmVersionType = "0.2.1"
	KpmAbiVersion_0_2_0 KpmVersionType = "0.2.0"
	KpmAbiVersion_0_1_0 KpmVersionType = "0.1.0"
)
