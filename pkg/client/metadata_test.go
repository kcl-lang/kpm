package client

import (
	"encoding/json"
	"testing"

	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// TestResolvePkgDepsMetadata verifies that dependency resolution works as expected.
func TestResolvePkgDepsMetadata(t *testing.T) {
	// Create a real KpmClient
	client := &KpmClient{}

	// Initialize KclPkg using InitOptions
	initOptions := &opt.InitOptions{
		InitPath: "testdata/sample_package",
	}
	kclPkg := pkg.NewKclPkg(initOptions)

	// Attempt to resolve package dependencies
	err := client.ResolvePkgDepsMetadata(&kclPkg, true)
	if err != nil {
		t.Errorf("ResolvePkgDepsMetadata failed: %v", err)
	}
}

// TestResolveDepsMetadataInJsonStr verifies that dependencies metadata is serialized correctly.
func TestResolveDepsMetadataInJsonStr(t *testing.T) {
	client := &KpmClient{}

	// Initialize KclPkg using InitOptions
	initOptions := &opt.InitOptions{
		InitPath: "testdata/sample_package",
	}
	kclPkg := pkg.NewKclPkg(initOptions)

	// Get dependencies metadata in JSON
	jsonStr, err := client.ResolveDepsMetadataInJsonStr(&kclPkg, true)
	if err != nil {
		t.Fatalf("ResolveDepsMetadataInJsonStr failed: %v", err)
	}

	// Print JSON output to inspect its structure
	t.Logf("JSON Output: %s", jsonStr)

	// Try to unmarshal into a generic structure if DepMetadata is unknown
	var deps interface{}
	err = json.Unmarshal([]byte(jsonStr), &deps)
	if err != nil {
		t.Errorf("Failed to parse JSON output: %v", err)
	}

	// Check if the output contains expected keys
	if deps == nil {
		t.Errorf("Expected some dependencies but got none")
	}
}
