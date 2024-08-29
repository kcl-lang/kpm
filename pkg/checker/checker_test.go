package checker

import (
	"testing"

	"github.com/elliotchance/orderedmap/v2"
	"gotest.tools/v3/assert"
	pkg "kcl-lang.io/kpm/pkg/package"
)

func TestDepCheckerCheck(t *testing.T) {
	depChecker := NewDepChecker(
		&IdentChecker{},
		&VersionChecker{},
		&SumChecker{},
	)
	deps1 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps1.Set("kcl1", pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Version:  "0.0.1",
		Sum:      "",
	})
	deps1.Set("kcl2", pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Version:  "0.2.1",
		Sum:      "",
	})

	deps2 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps2.Set("kcl1", pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Version:  "0.0.1",
		Sum:      "no-sum-check-enabled",
	})
	deps2.Set("kcl2", pkg.Dependency{
		Name:     "kcl2",
		FullName: "kcl2",
		Version:  "0.2.1",
		Sum:      "no-sum-check-enabled",
	})

	deps3 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps3.Set("kcl1", pkg.Dependency{
		Name:     ".kcl1",
		FullName: "kcl1",
		Version:  "0.0.1",
		Sum:      "",
	})

	deps4 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps4.Set("kcl1", pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Version:  "1.0.0-alpha#",
		Sum:      "",
	})

	deps5 := orderedmap.NewOrderedMap[string, pkg.Dependency]()
	deps5.Set("kcl1", pkg.Dependency{
		Name:     "kcl1",
		FullName: "kcl1",
		Version:  "0.0.1",
		Sum:      "invalid-no-sum-check-disabled",
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
			name: "Invalid kcl package - invalid dependency name",
			KclPkg: pkg.KclPkg{
				ModFile: pkg.ModFile{
					HomePath: "path/to/modfile",
				},
				HomePath: "path/to/kcl/pkg",
				Dependencies: pkg.Dependencies{
					Deps: deps3,
				},
				NoSumCheck: false,
			},
			wantErr: true,
		},
		{
			name: "Invalid kcl package - invalid dependency version",
			KclPkg: pkg.KclPkg{
				ModFile: pkg.ModFile{
					HomePath: "path/to/modfile",
				},
				HomePath: "path/to/kcl/pkg",
				Dependencies: pkg.Dependencies{
					Deps: deps4,
				},
				NoSumCheck: false,
			},
			wantErr: true,
		},
		{
			name: "Invalid kcl package - with no sum check disabled - checksum mismatches",
			KclPkg: pkg.KclPkg{
				ModFile: pkg.ModFile{
					HomePath: "path/to/modfile",
				},
				HomePath: "path/to/kcl/pkg",
				Dependencies: pkg.Dependencies{
					Deps: deps5,
				},
				NoSumCheck: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := depChecker.Check(tt.KclPkg)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("depChecker.Check(%v) = %v, want error %v", tt.KclPkg, gotErr, tt.wantErr)
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
		{"Valid Name - Simple", "myDependency", true},
		{"Valid Name - With Underscore", "my_dependency", true},
		{"Valid Name - With Hyphen", "my-dependency", true},
		{"Valid Name - With Dot", "my.dependency", true},
		{"Valid Name - Mixed Case", "MyDependency", true},
		{"Valid Name - Long Name", "My_Very-Long.Dependency", true},
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
