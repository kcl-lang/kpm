package client

import (
	"os"
	"testing"

	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
)

// TestPackagePkg verifies that the PackagePkg function correctly creates a .tar file.
func TestPackagePkg(t *testing.T) {
	client := &KpmClient{}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kclpkg_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Initialize KclPkg with a valid path
	initOptions := &opt.InitOptions{
		InitPath: tempDir,
	}
	kclPkg := pkg.NewKclPkg(initOptions)

	// Run the packaging function
	tarPath, err := client.PackagePkg(&kclPkg, false)
	if err != nil {
		t.Fatalf("PackagePkg failed: %v", err)
	}

	// Check if the .tar file was created
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Errorf("Expected tar file at %s but it was not created", tarPath)
	} else {
		t.Logf("Tar file created successfully at %s", tarPath)
	}
}

// TestPackage verifies that the Package function correctly archives the package.
func TestPackage(t *testing.T) {
	client := &KpmClient{}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kclpkg_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	// Initialize KclPkg
	initOptions := &opt.InitOptions{
		InitPath: tempDir,
	}
	kclPkg := pkg.NewKclPkg(initOptions)

	// Determine tar path
	tarPath := kclPkg.DefaultTarPath()

	// Ensure any existing tar file is removed before testing
	_ = os.Remove(tarPath)

	// Run the package function
	err = client.Package(&kclPkg, tarPath, false)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}

	// Verify the tar file was created
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		t.Errorf("Expected tar file at %s but it was not created", tarPath)
	} else {
		t.Logf("Tar file created successfully at %s", tarPath)
	}

	// Clean up after test
	_ = os.Remove(tarPath)
}
