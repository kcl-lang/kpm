# A Tour of various Selection Strategy

Almost in every Package Manager, there are 4 main actors:

**Project code** is the code for which we want to manage the dependency.

**Manifest file** is a file in which the dependencies for the project code are listed.

**Lock file** is a package manager generated file that contains all the information necessary to reproduce the same dependency tree across any platform.

**Dependency code** is the fetched code of the resolved dependencies.

To prevent dependency conflicts, dependency resolution and optimizing dependency tree, selection strategy is used by package manager.

Lets study the selection strategy of famous package managers: 

### Cargo

In rust, dependencies are specified in cargo.toml file in the format <name\> = <version\>. [Semver](https://semver.org/) is used when specifying version numbers. 

To update a dependency safely, rust uses the concept of version compatibility.

Cargo uses Semantic to constrain the compatibility between
different versions of a package. Cargo uses the leftmost nonzero number of the version to determine compatibility, e.g. version numbers 1.0.16 and 1.1.16 are considered compatible, and cargo considers it safe to update in the compatible range, but updates outside the compatibility range are not allowed

lets see how semantic version requirement is considered during resolution of dependencies.

• When multiple packages require a common dependency, the resolver aims to ensure they utilize the same version within a SemVer compatibility range, favoring the latest version within that range. For example, if package 1 depends on `foo = "1.0"` and package 2 depends on `foo = "1.1"`, then if the highest version during lock file generation is 1.2.1, both packages will utilize this version. Even if a new version like 2.0.0 is released later, it won't be automatically chosen as it's deemed incompatible.
<br>
• If multiple packages have a common dependency with semver-incompatible versions, then Cargo will allow this, but will build two separate copies of the dependency.
<br>
• If the resolver is constrained to two different versions within the same compatibility range, it will raise an error, as multiple versions within the range are not permitted.
<br>
• Many of the versions in Cargo are pre-releases, which Cargo does not usually use. To use these pre-releases, the user must specify the pre-release version, which often means that it is unstable.

Cargo's dependency parser considers various factors beyond Semantic Versioning requirements, including package characteristics, dependency types, parser versions, and numerous other rules.

Running `cargo build` will resolve dependencies listed in the manifest file and save the result in `cargo.lock` file. 

#### Advantages

• **Compatibility Assurance**: Cargo ensures that dependencies adhere to Semantic Versioning (SemVer) rules, promoting compatibility and reducing potential conflicts between packages.

• **Integration with Rust Ecosystem**: Cargo is tightly integrated with the Rust ecosystem, facilitating seamless dependency management for Rust projects. Its integration with tools like rustc, the Rust compiler, and rustup, the Rust toolchain installer, enhances developer productivity and simplifies the development workflow.

#### Disadvantages:

• **Security Risks in Package Ecosystem**: use of yanked values and unsafe keywords in real-world Rust libraries and applications contribute to these risks.

• **Dependency Bloat**: In some cases, Cargo's dependency resolution may result in the inclusion of unnecessary or overly large dependencies, leading to increased binary sizes or longer build times. This can impact the performance and efficiency of the final application, especially in resource-constrained environments.

### Go Package Manager

The Go package manager adopts a Minimum Version Selection (MVS) approach to determine which packages to include in the final list for building. MVS aims to create builds that closely mirror the dependencies used by the package author during development. This means that when a user builds a project, the dependencies chosen are as similar as possible to the ones the original author developed against.

Minimal Version Selection (MVS) operates on the assumption that each module specifies only the minimum versions of its dependencies, adhering to the import compatibility rule where newer versions are expected to be compatible with older ones. This means dependency requirements include only minimum versions, without specifying maximum versions or incompatible later versions.

version selection strategy is meant to provide algorithms for four operations on build list:

1. Construct the current build list: 
    
    The rough build list for package M would be just the list of all modules reachable in the requirement graph starting at M and following arrows. This can be accomplished through a straightforward recursive traversal of the graph, ensuring to skip nodes that have already been visited. The rough built list can then be converted to the final build list.

2. Upgrade all modules to their latest versions:

    This can be achieved by running `go get -u` which will upgrade all the modules to their latest versions. 
    Upgrading the modules would mean all arrows in the dependency graph is now pointing to the latest version of the modules. This will result in a upgraded dependency graph but changes in the dependency graph alone won't cause future builds to use the updated modules. To achieve this we need a change in our built list in a way that won't affect dependent packages built list, as upgrades should be limited to our package alone.

    At first glance, it would seem intutive to include all the updated packages in our built list. But, not all packages are necessary and we want to include as few additional modules as possible. To produce a minimum requirement list, an helper algorithm R is introduced.

    **Algorithm R**: 

    To compute a minimal requirement list inducing a given build list below the target, reverse postorder traversal is employed, ensuring modules are visited after all those pointing into them. Each module is added only if it's not implied by previously visited ones. 

3. Upgrade one module to a specific newer version:

    Upgrading all modules to their latest versions can be risky, so developers often opt to upgrade only one module. 

    Upgrading one module mean that the arrow which earlier pointed to that module is now pointing to the upgraded version. We can construct a built list from the updated dependency graph, which can then be fed to Algorithm R to get a minimum requirement list.

4. Downgrade one module to a specific older version.

    The downgrade algorithm examines each of the target's requirements separately. If a requirement conflicts with the proposed downgrade, meaning its build list contains a version of a module that is no longer allowed, the algorithm iterates through older versions until finding one that aligns with the downgrade.

    Downgrades make changes to the built list by removing requirements.
