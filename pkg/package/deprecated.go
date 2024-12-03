package pkg

import (
	"kcl-lang.io/kpm/pkg/settings"
	"kcl-lang.io/kpm/pkg/utils"
)

// FillDepInfo will fill registry information for a dependency.
// Deprecated: this function is not used anymore.
func (dep *Dependency) FillDepInfo(homepath string) error {
	if dep.Source.Oci != nil {
		settings := settings.GetSettings()
		if settings.ErrorEvent != nil {
			return settings.ErrorEvent
		}
		if dep.Source.Oci.Reg == "" {
			dep.Source.Oci.Reg = settings.DefaultOciRegistry()
		}

		if dep.Source.Oci.Repo == "" {
			urlpath := utils.JoinPath(settings.DefaultOciRepo(), dep.Name)
			dep.Source.Oci.Repo = urlpath
		}
	}
	if dep.Source.Local != nil {
		dep.LocalFullPath = dep.Source.Local.Path
	}

	dep.FullName = dep.GenDepFullName()

	return nil
}

// FillDependenciesInfo will fill registry information for all dependencies in a kcl.mod.
// Deprecated: this function is not used anymore.
func (modFile *ModFile) FillDependenciesInfo() error {
	for _, k := range modFile.Deps.Keys() {
		v, ok := modFile.Deps.Get(k)
		if !ok {
			break
		}
		err := v.FillDepInfo(modFile.HomePath)
		if err != nil {
			return err
		}
		modFile.Deps.Set(k, v)
	}
	return nil
}
