// Package features sets the feature gates that
// kpm uses to enable or disable certain features.
package features

import (
	"os"
	"strings"
	"sync"
)

const (
	// SupportMVS is the feature gate for enabling the support for MVS.
	SupportMVS = "SupportMVS"
	// SupportNewStorage is the feature gate for enabling the support for the new storage structure.
	SupportNewStorage = "SupportNewStorage"
)

var (
	features = map[string]bool{
		SupportMVS:        false,
		SupportNewStorage: false,
	}
	mu sync.Mutex
)

func init() {
	// supports the env: export KPM_FEATURE_GATES="SupportMVS=true"
	envVar := os.Getenv("KPM_FEATURE_GATES")
	if envVar != "" {
		parseFeatureGates(envVar)
	}
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
	mu.Lock()
	defer mu.Unlock()
	if enabled, ok := features[feature]; ok {
		return enabled, nil
	}
	return false, nil
}

// Enable enables the specified feature. If the feature is not
// present, it's a no-op.
func Enable(feature string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := features[feature]; ok {
		features[feature] = true
	}
}

// Disable disables the specified feature. If the feature is not
// present, it's a no-op.
func Disable(feature string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := features[feature]; ok {
		features[feature] = false
	}
}

// parseFeatureGates parses the feature gates from the given
func parseFeatureGates(envVar string) {
	pairs := strings.Split(envVar, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1]) == "true"
		if _, ok := features[key]; ok {
			features[key] = value
		}
	}
}
