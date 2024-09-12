/*
Copyright 2022 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package features sets the feature gates that
// kpm uses to enable or disable certain features.
package features

import feathelper "github.com/fluxcd/pkg/runtime/features"

const (
	// SupportMVS is the feature gate for enabling the support for MVS.
	SupportMVS = "SupportMVS"
)

var features = map[string]bool{
	SupportMVS: false,
}

// FeatureGates contains a list of all supported feature gates and
// their default values.
func FeatureGates() map[string]bool {
	return features
}

// Enabled verifies whether the feature is enabled or not.
//
// This is only a wrapper around the Enabled func in
// pkg/runtime/features, so callers won't need to import
// both packages for checking whether a feature is enabled.
func Enabled(feature string) (bool, error) {
	return feathelper.Enabled(feature)
}

// Enable enables the specified feature. If the feature is not
// present, it's a no-op.
func Enable(feature string) {
	if _, ok := features[feature]; ok {
		features[feature] = true
	}
}

// Disable disables the specified feature. If the feature is not
// present, it's a no-op.
func Disable(feature string) {
	if _, ok := features[feature]; ok {
		features[feature] = false
	}
}
