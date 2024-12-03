package client

import (
	"encoding/json"

	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/reporter"
)

// ResolveDepsMetadata will calculate the local storage path of the external package,
// and check whether the package exists locally.
// If the package does not exist, it will re-download to the local.
// Since redownloads are not triggered if local dependencies exists,
// indirect dependencies are also synchronized to the lock file by `lockDeps`.
func (c *KpmClient) ResolvePkgDepsMetadata(kclPkg *pkg.KclPkg, update bool) error {
	var err error
	if kclPkg.IsVendorMode() {
		err = c.VendorDeps(kclPkg)
	} else {
		_, err = c.Update(
			WithUpdatedKclPkg(kclPkg),
			WithOffline(!update),
		)
	}
	return err
}

// ResolveDepsMetadataInJsonStr will calculate the local storage path of the external package,
// and check whether the package exists locally. If the package does not exist, it will re-download to the local.
// Finally, the calculated metadata of the dependent packages is serialized into a json string and returned.
func (c *KpmClient) ResolveDepsMetadataInJsonStr(kclPkg *pkg.KclPkg, update bool) (string, error) {
	// 1. Calculate the dependency path, check whether the dependency exists
	// and re-download the dependency that does not exist.
	err := c.ResolvePkgDepsMetadata(kclPkg, update)
	if err != nil {
		return "", err
	}

	// 2. Serialize to JSON
	depMetadatas, err := kclPkg.GetDepsMetadata()
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(&depMetadatas)
	if err != nil {
		return "", reporter.NewErrorEvent(reporter.Bug, err, "internal bug: failed to marshal the dependencies into json")
	}

	return string(jsonData), nil
}
