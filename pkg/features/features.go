// Package features sets the feature gates that
// kpm uses to enable or disable certain features.
package features

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
	if enabled, ok := features[feature]; ok {
		return enabled, nil
	}
	return false, nil
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
