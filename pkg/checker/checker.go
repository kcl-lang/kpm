package checker

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/go-version"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"

	"kcl-lang.io/kpm/pkg/constants"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/oci"
	"kcl-lang.io/kpm/pkg/opt"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// Checker defines an interface for checking KclPkg dependencies.
type Checker interface {
	Check(pkg.KclPkg) error
}

// ModChecker is responsible for running multiple checkers on a KCL module.
type ModChecker struct {
	checkers []Checker
}

// ModCheckerOption configures how we set up ModChecker.
type ModCheckerOption func(*ModChecker)

// NewModChecker creates a new ModChecker with options.
func NewModChecker(options ...ModCheckerOption) *ModChecker {
	ModChecker := &ModChecker{}
	for _, opt := range options {
		opt(ModChecker)
	}
	return ModChecker
}

// WithChecker adds a single Checker to ModChecker.
func WithChecker(checker Checker) ModCheckerOption {
	return func(c *ModChecker) {
		if c.checkers == nil {
			c.checkers = []Checker{}
		}
		c.checkers = append(c.checkers, checker)
	}
}

// WithCheckers adds multiple Checkers to ModChecker.
func WithCheckers(checkers ...Checker) ModCheckerOption {
	return func(c *ModChecker) {
		if c.checkers == nil {
			c.checkers = []Checker{}
		}
		c.checkers = append(c.checkers, checkers...)
	}
}

func (mc *ModChecker) AddChecker(checker Checker) {
	mc.checkers = append(mc.checkers, checker)
}

func (mc *ModChecker) CheckersSize() int {
	if mc.checkers == nil {
		return 0
	}
	return len(mc.checkers)
}

// Check runs all individual checks for a kclPkg.
func (mc *ModChecker) Check(kclPkg pkg.KclPkg) error {
	for _, checker := range mc.checkers {
		if err := checker.Check(kclPkg); err != nil {
			return err
		}
	}
	return nil
}

// IdentChecker validates the dependencies name in kclPkg.
type IdentChecker struct{}

// NewIdentChecker creates a new IdentChecker.
func NewIdentChecker() *IdentChecker {
	return &IdentChecker{}
}

func (ic *IdentChecker) Check(kclPkg pkg.KclPkg) error {
	if !isValidDependencyName(kclPkg.ModFile.Pkg.Name) {
		return fmt.Errorf("invalid name: %s", kclPkg.ModFile.Pkg.Name)
	}
	return nil
}

// VersionChecker validates the dependencies version in kclPkg.
type VersionChecker struct{}

// NewVersionChecker creates a new VersionChecker.
func NewVersionChecker() *VersionChecker {
	return &VersionChecker{}
}

func (vc *VersionChecker) Check(kclPkg pkg.KclPkg) error {
	if !isValidDependencyVersion(kclPkg.ModFile.Pkg.Version) {
		return fmt.Errorf("invalid version: %s for %s",
			kclPkg.ModFile.Pkg.Version, kclPkg.ModFile.Pkg.Name)
	}

	return nil
}

// SumChecker validates the dependencies sum in kclPkg.
type SumChecker struct {
	settings settings.Settings
}

// SumCheckerOption configures how we set up SumChecker.
type SumCheckerOption func(*SumChecker)

// NewSumChecker creates a new SumChecker with options.
func NewSumChecker(options ...SumCheckerOption) *SumChecker {
	sumChecker := &SumChecker{}
	for _, opt := range options {
		opt(sumChecker)
	}
	return sumChecker
}

// WithSettings sets the settings for SumChecker.
func WithSettings(settings settings.Settings) SumCheckerOption {
	return func(s *SumChecker) {
		s.settings = settings
	}
}

// Check verifies the checksums of the dependencies in the KclPkg.
func (sc *SumChecker) Check(kclPkg pkg.KclPkg) error {
	if kclPkg.NoSumCheck {
		return nil
	}

	for _, key := range kclPkg.Dependencies.Deps.Keys() {
		dep, _ := kclPkg.Dependencies.Deps.Get(key)
		trustedSum, err := sc.getTrustedSum(dep)
		if err != nil {
			return fmt.Errorf("failed to get checksum from trusted source: %w", err)
		}
		if dep.Sum != trustedSum {
			return fmt.Errorf("checksum verification failed for '%s': expected '%s', got '%s'", dep.Name, trustedSum, dep.Sum)
		}
	}
	return nil
}

// isValidDependencyName checks whether the given dependency name is valid.
func isValidDependencyName(name string) bool {
	validNamePattern := `^[a-z][a-z0-9_]*(?:-[a-z0-9_]+)*$`
	regex := regexp.MustCompile(validNamePattern)
	return regex.MatchString(name)
}

// isValidDependencyVersion checks whether the given version is a valid semantic version string.
func isValidDependencyVersion(v string) bool {
	_, err := version.NewVersion(v)
	return err == nil
}

// getTrustedSum retrieves the trusted checksum for the given dependency.
func (sc *SumChecker) getTrustedSum(dep pkg.Dependency) (string, error) {
	if dep.Source.Oci == nil {
		return "", fmt.Errorf("dependency is not from OCI")
	}

	sc.populateOciFields(dep)

	manifest, err := sc.fetchOciManifest(dep)
	if err != nil {
		return "", err
	}

	return sc.extractChecksumFromManifest(manifest)
}

// populateOciFields fills in missing OCI fields with default values from settings.
func (sc *SumChecker) populateOciFields(dep pkg.Dependency) {
	if len(dep.Source.Oci.Reg) == 0 {
		dep.Source.Oci.Reg = sc.settings.DefaultOciRegistry()
	}

	if len(dep.Source.Oci.Repo) == 0 {
		dep.Source.Oci.Repo = utils.JoinPath(sc.settings.DefaultOciRepo(), dep.Name)
	}
}

// fetchOciManifest retrieves the OCI manifest for the given dependency.
func (sc *SumChecker) fetchOciManifest(dep pkg.Dependency) (ocispec.Manifest, error) {
	manifest := ocispec.Manifest{}
	jsonDesc, err := sc.FetchOciManifestIntoJsonStr(opt.OciFetchOptions{
		FetchBytesOptions: oras.DefaultFetchBytesOptions,
		OciOptions: opt.OciOptions{
			Reg:  dep.Source.Oci.Reg,
			Repo: dep.Source.Oci.Repo,
			Tag:  dep.Source.Oci.Tag,
		},
	})
	if err != nil {
		return manifest, reporter.NewErrorEvent(reporter.FailedFetchOciManifest, err, fmt.Sprintf("failed to fetch the manifest of '%s'", dep.Name))
	}

	err = json.Unmarshal([]byte(jsonDesc), &manifest)
	if err != nil {
		return manifest, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return manifest, nil
}

// FetchOciManifestIntoJsonStr fetches the OCI manifest and returns it as a JSON string.
func (sc *SumChecker) FetchOciManifestIntoJsonStr(opts opt.OciFetchOptions) (string, error) {
	repoPath := utils.JoinPath(opts.Reg, opts.Repo)
	cred, err := sc.GetCredentials(opts.Reg)
	if err != nil {
		return "", err
	}

	ociCli, err := oci.NewOciClientWithOpts(
		oci.WithCredential(cred),
		oci.WithRepoPath(repoPath),
		oci.WithSettings(&sc.settings),
	)
	if err != nil {
		return "", err
	}

	manifestJson, err := ociCli.FetchManifestIntoJsonStr(opts)
	if err != nil {
		return "", err
	}
	return manifestJson, nil
}

// GetCredentials retrieves the OCI credentials for the given hostname.
func (sc *SumChecker) GetCredentials(hostName string) (*remoteauth.Credential, error) {
	credCli, err := downloader.LoadCredentialFile(sc.settings.CredentialsFile)
	if err != nil {
		return nil, err
	}

	creds, err := credCli.Credential(hostName)
	if err != nil {
		return nil, err
	}

	return creds, nil
}

// extractChecksumFromManifest extracts the checksum from the OCI manifest.
func (sc *SumChecker) extractChecksumFromManifest(manifest ocispec.Manifest) (string, error) {
	if value, ok := manifest.Annotations[constants.DEFAULT_KCL_OCI_MANIFEST_SUM]; ok {
		return value, nil
	}
	return "", fmt.Errorf("checksum annotation not found in manifest")
}
