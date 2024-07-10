package constants

const (
	Default             = "default"
	DefaultOciScheme    = "default-oci"
	KFilePathSuffix     = ".k"
	TarPathSuffix       = ".tar"
	TgzPathSuffix       = ".tgz"
	GitPathSuffix       = ".git"
	OciScheme           = "oci"
	GitScheme           = "git"
	HttpScheme          = "http"
	HttpsScheme         = "https"
	SshScheme           = "ssh"
	FileEntry           = "file"
	FileWithKclModEntry = "file_with_kcl_mod"
	UrlEntry            = "url"
	RefEntry            = "ref"
	TarEntry            = "tar"
	GitEntry            = "git"

	GitBranch = "branch"
	GitCommit = "commit"

	Tag = "tag"

	KCL_MOD                              = "kcl.mod"
	KCL_MOD_LOCK                         = "kcl.mod.lock"
	KCL_YAML                             = "kcl.yaml"
	OCI_SEPARATOR                        = ":"
	KCL_PKG_TAR                          = "*.tar"
	DEFAULT_KCL_FILE_NAME                = "main.k"
	DEFAULT_KCL_FILE_CONTENT             = "The_first_kcl_program = 'Hello World!'"
	DEFAULT_KCL_OCI_MANIFEST_NAME        = "org.kcllang.package.name"
	DEFAULT_KCL_OCI_MANIFEST_VERSION     = "org.kcllang.package.version"
	DEFAULT_KCL_OCI_MANIFEST_DESCRIPTION = "org.kcllang.package.description"
	DEFAULT_KCL_OCI_MANIFEST_SUM         = "org.kcllang.package.sum"
	DEFAULT_CREATE_OCI_MANIFEST_TIME     = "org.opencontainers.image.created"
	URL_PATH_SEPARATOR                   = "/"
	LATEST                               = "latest"

	// The pattern of the external package argument.
	EXTERNAL_PKGS_ARG_PATTERN = "%s=%s"

	// The pattern of the git url
	GIT_PROTOCOL_URL_PATTERN = "?ref=%s"
)
