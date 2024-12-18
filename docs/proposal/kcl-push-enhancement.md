### Enhancement Proposal: Integration of Validation Checks in `kpm push`

**Author:** [SkySingh04](https://linktr.ee/skysingh04)

---

### **Overview**
This proposal aims to integrate validation checks for the `kcl.mod` file during the `push` operation in the `kpm` tool. By leveraging the existing `Checker` module, we ensure that fields like `name` and `version` are validated before pushing a package to the OCI registry. The objective is to maintain package integrit and prevent incomplete uploads, following the best practices observed in other package managers.


### **Background**
The `push` command in `kpm` is used to upload KCL packages to an OCI (Open Container Initiative) registry. Currently, it lacks validation for mandatory fields such as `name` and `version` in the `kcl.mod` file, which can result in inconsistent or incomplete packages being uploaded. Addressing this issue will enhance package management and ensure consistency across the registry.

As pointed out by @zong-zhe in this [comment](https://github.com/kcl-lang/kpm/pull/562#issuecomment-2533751341), the existing `Checker` module already performs checks for:
- `name`
- `version`
- `checksum`

This proposal outlines how the `Checker` can be seamlessly integrated into the `push` workflow while also suggesting further enhancements inspired by other package managers like Cargo, Go Modules, and Helm.

---

### **Proposed Solution**

#### **1. Integration of the `Checker` Module**
The `Checker` module will be integrated as a pre-push validation step to ensure that packages meet all necessary criteria before being uploaded. This will involve the following steps:

- **Loading the `kcl.mod` File:**
  - The `pushCurrentPackage` and `pushTarPackage` functions will load the `kcl.mod` file.

- **Validation Hook:**
  - Pass the loaded `kcl.mod` file to the `Checker` for validation. If the `Checker` detects missing or invalid fields, the `push` operation will be halted and error would be thrown.

#### **2. Error Messaging**
The validation process will surface user-friendly error messages, guiding users to resolve issues before retrying the `push` operation. 

Some examples include but are not limited to :

- **Missing Fields:**  Indicate which mandatory fields (e.g., `name`, `version`) are missing from the `kcl.mod` file.
- **Invalid Values:** Specify any fields that contain invalid values, such as incorrect version formats or unsupported characters.
- **Checksum Mismatches:** Alert users if the checksum does not match the expected value.

These error messages are standard industry practice as noticed in other package manager tools.

#### **3. Code Integration Example**
The following pseudo-code demonstrates how the `Checker` will be integrated:

```go
func pushCurrentPackage(pkgPath string) error {
    // Load the kcl.mod file
    modFile, err := loadKclMod(pkgPath)
    if err != nil {
        return fmt.Errorf("Failed to load kcl.mod: %v", err)
    }

    // Validate the kcl.mod file using Checker
    if err := checker.CheckKclMod(modFile); err != nil {
        return fmt.Errorf("Validation Error: %v", err)
    }

    // Proceed with push if validation passes
    return uploadToRegistry(pkgPath, modFile)
}
```

### **Research Insights from Other Package Managers**

#### **1. Cargo**
Cargo performs metadata checks and ensures semantic versioning compliance. It validates fields like `name`, `version`, `description`, and `license` in the `Cargo.toml` file. Additionally, Cargo resolves dependencies and verifies checksums. [Source](https://doc.rust-lang.org/cargo/)

#### **2. Go Modules**
Go Modules validate the `module` name, enforce semantic versioning, and verify checksums against a public database. They also check the validity of dependencies. [Source](https://golang.org/ref/mod)

#### **3. Helm**
Helm performs linting and metadata validation for `Chart.yaml` files. It ensures the completeness of package manifests and validates dependencies. [Source](https://helm.sh/docs/topics/chart_best_practices/)

#### **4. NPM**
NPM also validates the package `name`, `version`, `description`, and `license`. It ensures the package complies with semantic versioning and performs integrity checks by verifying checksums against the npm registry. [Source](https://docs.npmjs.com/cli/v7/configuring-npm/package-json)

#### **5. Maven**
Maven checks that the `groupId`, `artifactId`, and `version` in the `pom.xml` file are correctly specified. It validates the dependencies and ensures they match the required versions and supports metadata verification and checksums for security. [Source](https://maven.apache.org/guides/introduction/introduction-to-the-pom.html)


Here's the revised proposal, starting from the point marked:

---

#### **Proposed Improvements to `kpm` Based on the Above Research**

1. **Introduce a `kpm lint` Command**  
   Add a `lint` command to validate the structure and metadata of KCL packages before pushing. This feature will allow developers to proactively identify and resolve issues in their packages.

2. **Add Optional Metadata Fields**  
   Expand the `kcl.mod` schema to include optional fields such as:
   - `description`: A brief summary of the package.
   - `license`: License information for the package.
   - `authors`: A list of contributors or maintainers.
   These fields improve package documentation and traceability.

3. **Enforce Semantic Versioning**  
   Validate the `version` field to ensure compliance with the `major.minor.patch` format defined by semantic versioning standards. This standardization will enhance dependency management and version resolution.

4. **Implement Checksum Verification**  
   Include checksum validation for uploaded packages to ensure integrity and prevent tampering. This feature would compare the locally generated checksum with a stored value in the registry.

5. **Validate Dependencies (Future Feature)**  
   When dependencies are introduced in KCL packages, the `Checker` module should validate:
   - The presence and validity of dependency metadata.
   - Compatibility of dependencies with the package being pushed.


### **Conclusion**

By integrating the `Checker` module into the `push` workflow, enhancing its capabilities, and adopting best practices from other package managers, this proposal strengthens `kpm` as a reliable package management tool. These changes will improve the integrity, usability, and scalability of KCL packages while setting the foundation for future enhancements.
The proposed solution ensures package integrity by validating standards before upload, enhances developer experience with clear error messages and a `lint` command, aligns `kpm` with industry standards from tools like Cargo and NPM, and provides a scalable validation framework for future needs.
