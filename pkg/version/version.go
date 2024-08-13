// Copyright 2022 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

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
	KpmAbiVersion         KpmVersionType = KpmAbiVersion_0_10_0
	KpmVersionType_latest                = KpmAbiVersion_0_10_0

	KpmAbiVersion_0_10_0 KpmVersionType = "0.10.0"
	KpmAbiVersion_0_9_0  KpmVersionType = "0.9.0"
	KpmAbiVersion_0_8_0  KpmVersionType = "0.8.0"
	KpmAbiVersion_0_7_0  KpmVersionType = "0.7.0"
	KpmAbiVersion_0_6_0  KpmVersionType = "0.6.0"
	KpmAbiVersion_0_5_0  KpmVersionType = "0.5.0"
	KpmAbiVersion_0_4_7  KpmVersionType = "0.4.7"
	KpmAbiVersion_0_4_6  KpmVersionType = "0.4.6"
	KpmAbiVersion_0_4_5  KpmVersionType = "0.4.5"
	KpmAbiVersion_0_4_4  KpmVersionType = "0.4.4"
	KpmAbiVersion_0_4_3  KpmVersionType = "0.4.3"
	KpmAbiVersion_0_4_2  KpmVersionType = "0.4.2"
	KpmAbiVersion_0_4_1  KpmVersionType = "0.4.1"
	KpmAbiVersion_0_4_0  KpmVersionType = "0.4.0"
	KpmAbiVersion_0_3_7  KpmVersionType = "0.3.7"
	KpmAbiVersion_0_3_6  KpmVersionType = "0.3.6"
	KpmAbiVersion_0_3_5  KpmVersionType = "0.3.5"
	KpmAbiVersion_0_3_4  KpmVersionType = "0.3.4"
	KpmAbiVersion_0_3_3  KpmVersionType = "0.3.3"
	KpmAbiVersion_0_3_2  KpmVersionType = "0.3.2"
	KpmAbiVersion_0_3_1  KpmVersionType = "0.3.1"
	KpmAbiVersion_0_3_0  KpmVersionType = "0.3.0"
	KpmAbiVersion_0_2_6  KpmVersionType = "0.2.6"
	KpmAbiVersion_0_2_5  KpmVersionType = "0.2.5"
	KpmAbiVersion_0_2_4  KpmVersionType = "0.2.4"
	KpmAbiVersion_0_2_3  KpmVersionType = "0.2.3"
	KpmAbiVersion_0_2_2  KpmVersionType = "0.2.2"
	KpmAbiVersion_0_2_1  KpmVersionType = "0.2.1"
	KpmAbiVersion_0_2_0  KpmVersionType = "0.2.0"
	KpmAbiVersion_0_1_0  KpmVersionType = "0.1.0"
)
