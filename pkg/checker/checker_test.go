package checker

import (
	"runtime"
	"testing"

	"github.com/elliotchance/orderedmap/v2"
	"gotest.tools/v3/assert"

	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/mock"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/test"
)

func TestModCheckerCheck(t *testing.T) {
	ModChecker := NewModChecker(WithCheckers(NewIdentChecker(), NewVersionChecker(), NewSumChecker()))

	deps1 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps1.Set("kcl1", pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Version:  "0.0.1",
		Sum:      "no-sum-check-enabled",
	})
	deps1.Set("kcl2", pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Version:  "0.2.1",
		Sum:      "no-sum-check-enabled",
	})

	tests := []struct {
		name    string
		KclPkg  pkg.KclPkg
		wantErr bool
	}{
		{
			name: "valid kcl package - with no sum check enabled",
			KclPkg: pkg.KclPkg{
				ModFile: pkg.ModFile{
					Pkg: pkg.Package{
						Name:    "testmod",
						Version: "0.0.1",
					},
					HomePath: "path/to/modfile",
				},
				HomePath: "path/to/kcl/pkg",
				Dependencies: pkg.Dependencies{
					Deps: deps1,
				},
				NoSumCheck: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ModChecker.Check(tt.KclPkg)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ModChecker.Check(%v) = %v, want error %v", tt.KclPkg, gotErr, tt.wantErr)
			}
		})
	}
}

func TestIsValidDependencyName(t *testing.T) {
	tests := []struct {
		name           string
		dependencyName string
		want           bool
	}{
		{"Empty Name", "", false},
		{"Valid Name - Simple", "myDependency", false},
		{"Valid Name - With Underscore", "my_dependency", true},
		{"Valid Name - With Hyphen", "my-dependency", true},
		{"Valid Name - With Dot", "my.dependency", false},
		{"Valid Name - Mixed Case", "MyDependency", false},
		{"Valid Name - Long Name", "My_Very-Long.Dependency", false},
		{"Contains Number", "depend3ncy", true},
		{"Starts with Special Character", "-dependency", false},
		{"Starts and Ends with Dot", ".dependency.", false},
		{"Starts and Ends with Hyphen", "-dependency-", false},
		{"Ends with Special Character", "dependency-", false},
		{"Only Special Characters", "._-", false},
		{"Some Special Characters", "dep!@#$%", false},
		{"Whitespace", "my dependency", false},
		{"Leading Whitespace", " dependency", false},
		{"Trailing Whitespace", "dependency ", false},
		{"Only Dot", ".", false},
		{"Only Hyphen", "-", false},
		{"Only Underscore", "_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDependencyName(tt.dependencyName)
			assert.Equal(t, got, tt.want, tt.dependencyName)
		})
	}
}

func TestIsValidDependencyVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"Empty String in version", "", false},
		{"Valid SemVer - Major", "1", true},
		{"Valid SemVer - Major.Minor", "1.0", true},
		{"Valid SemVer - Major.Minor.Patch", "1.0.0", true},
		{"Valid SemVer with Pre-release", "1.0.0-alpha", true},
		{"Invalid Pre-release Format", "1.0.0-alpha..1", false},
		{"Invalid Characters in Pre-release", "1.0.0-alpha#", false},
		{"Valid SemVer with Pre-release and Numeric", "1.0.0-alpha.1", true},
		{"Valid SemVer with Build Metadata", "1.0.0+001", true},
		{"Valid SemVer with Pre-release and Build Metadata", "1.0.0-beta+exp.sha.5114f85", true},
		{"Valid SemVer - Major.Minor.Patch with Leading Zeros", "01.02.03", true},
		{"Trailing Dot in version", "1.0.", false},
		{"Leading Dot in version", ".1.0", false},
		{"Valid SemVer - Too Many Dots", "1.0.0.0", true},
		{"Characters Only", "abc", false},
		{"Valid SemVer - Mixed Characters", "1.0.0abc", true},
		{"Whitespace", "1.0.0 ", false},
		{"Valid SemVer - Non-Numeric", "v1.0.0", true},
		{"Special Characters", "!@#$%", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDependencyVersion(tt.version)
			assert.Equal(t, got, tt.want, tt.version)
		})
	}
}

func getTestSettings() (*settings.Settings, error) {
	settings := settings.GetSettings()

	if settings.ErrorEvent != (*reporter.KpmEvent)(nil) {
		return nil, settings.ErrorEvent
	}
	return settings, nil
}

func TestModCheckerCheck_WithTrustedSum(t *testing.T) {
	testFunc := func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping TestModCheckerCheck_WithTrustedSum test on Windows")
		}

		// Start the local Docker registry required for testing
		err := mock.StartDockerRegistry()
		assert.Equal(t, err, nil)

		// Push the test package to the local OCI registry
		err = mock.PushTestPkgToRegistry()
		assert.Equal(t, err, nil)

		// Initialize settings for use with the ModChecker
		settings, err := getTestSettings()
		assert.Equal(t, err, nil)

		// Initialize the ModChecker with required checkers
		ModChecker := NewModChecker(WithCheckers(NewIdentChecker(), NewVersionChecker(), NewSumChecker(WithSettings(*settings))))

		deps1 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
		deps1.Set("kcl1", pkg.Dependency{
			Name:     "test_data",
			FullName: "test_data",
			Version:  "0.0.1",
			Sum:      "RpZZIvrXwfn5dpt6LqBR8+FlPE9Y+BEou47L3qaCCqk=",
			Source: downloader.Source{
				Oci: &downloader.Oci{
					Reg:  "localhost:5001",
					Repo: "test",
					Tag:  "0.0.1",
				},
			},
		})

		deps2 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
		deps2.Set("kcl1", pkg.Dependency{
			Name:     "test_data",
			FullName: "test_data",
			Version:  "0.0.1",
			Sum:      "Invalid-sum",
			Source: downloader.Source{
				Oci: &downloader.Oci{
					Reg:  "localhost:5001",
					Repo: "test",
					Tag:  "0.0.1",
				},
			},
		})

		tests := []struct {
			name    string
			KclPkg  pkg.KclPkg
			wantErr bool
		}{
			{
				name: "valid kcl package - with sum check",
				KclPkg: pkg.KclPkg{
					ModFile: pkg.ModFile{
						Pkg: pkg.Package{
							Name:    "testmod",
							Version: "0.0.1",
						},
						HomePath: "path/to/modfile",
					},
					HomePath: "path/to/kcl/pkg",
					Dependencies: pkg.Dependencies{
						Deps: deps1,
					},
					NoSumCheck: false,
				},
				wantErr: false,
			},
			{
				name: "valid kcl package - with no sum check enabled",
				KclPkg: pkg.KclPkg{
					ModFile: pkg.ModFile{
						Pkg: pkg.Package{
							Name:    "testmod",
							Version: "0.0.1",
						},
						HomePath: "path/to/modfile",
					},
					HomePath: "path/to/kcl/pkg",
					Dependencies: pkg.Dependencies{
						Deps: deps2,
					},
					NoSumCheck: true,
				},
				wantErr: false,
			},
			{
				name: "Invalid kcl package - with no sum check disabled - checksum mismatches",
				KclPkg: pkg.KclPkg{
					ModFile: pkg.ModFile{
						Pkg: pkg.Package{
							Name:    "testmod",
							Version: "0.0.1",
						},
						HomePath: "path/to/modfile",
					},
					HomePath: "path/to/kcl/pkg",
					Dependencies: pkg.Dependencies{
						Deps: deps2,
					},
					NoSumCheck: false,
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gotErr := ModChecker.Check(tt.KclPkg)
				if (gotErr != nil) != tt.wantErr {
					t.Errorf("ModChecker.Check(%v) = %v, want error %v", tt.KclPkg, gotErr, tt.wantErr)
				}
			})
		}

		// Clean the environment after all tests have been run
		err = mock.CleanTestEnv()
		assert.Equal(t, err, nil)
	}

	test.RunTestWithGlobalLock(t, "TestModCheckerCheck_WithTrustedSum", testFunc)
}
