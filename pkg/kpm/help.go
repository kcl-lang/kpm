package kpm

const (
	CliHelp         = `kpm  <command> [arguments]...`
	KclvmAbiVersion = "v0.4.3"
)
const DefaultKclModContent = string(`[expected]
kclvm_version="` + KclvmAbiVersion + `"`)

const DefaultRegistryAddr = "https://kpm.kusionstack.io"
const DefaultGitignore = "/external/\n"
