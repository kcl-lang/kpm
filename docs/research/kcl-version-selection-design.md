<h2>KCL version selection strategy</h2>

**Author**:  Akash Kumar

**Abstract**

In all the famous programming languages like go, rust, we cannot have two binaries of the same package with different versions in our system. Because of this, these languages require a selection strategy. But KCL is not a general language and kcl packages are not intended to be executed as a binary, that?s why currently multiple versions of the same package are allowed to reside in the file system. Currently, it seems no version selection should be required.

Before emphasizing on why a selection strategy is still needed, let's first talk about semantic versioning:

Semantic versioning (SemVer) is a widely adopted versioning scheme for software that aims to communicate changes in a clear and standardized way. In SemVer, version numbers are composed of three parts: MAJOR.MINOR.PATCH. Major versions signify significant changes, minor versions denote backward-compatible feature additions, and patch versions indicate backward-compatible bug fixes. Crucially, SemVer dictates that once a version is released, subsequent updates within the same major version should not introduce breaking changes. This ensures that users can safely upgrade to newer versions without fear of their software breaking due to unexpected changes. By adhering to SemVer principles, developers can maintain compatibility and predictability in their software releases, fostering smoother adoption and integration for end users.

All the packages in the kcl ecosystem follow semantic versioning. This means that if two or more versions of the same package is required somewhere in the dependency graph, given the versions don?t differ in MAJOR, then it seems intuitive to only include one version(later) of the package as there is no breaking change between them. This is the reason why we need a selection strategy.

We propose a minimum version selection strategy for the kpm package manager. Having a minimum version selection strategy would mean that we can have *high-fidelity builds*, in which the dependencies a user builds are as close as possible to the ones the author developed against.

**Background**

Here are the shortcomings of the current package manager.

Case 1:

Currently if a package has two dependencies which point to two different versions of the same package (Refer to the below dependency graph for clarity) then both versions of the package get downloaded.

![](/docs/research/dep-graph.png)

Case 2:

Currently, if a package is already present in local then also it will redownload it.


```bash
$  ls -d /home/akash/.kcl/kpm/k8s*
/home/akash/.kcl/kpm/k8s       /home/akash/.kcl/kpm/k8s_1.27
/home/akash/.kcl/kpm/k8s_1.14  /home/akash/.kcl/kpm/k8s_1.28
/home/akash/.kcl/kpm/k8s_1.17  /home/akash/.kcl/kpm/k8s_1.29
```
```bash
$  kcl mod add k8s
adding dependency 'k8s'
the lastest version '1.29' will be added
downloading 'kcl-lang/k8s:1.29' from 'ghcr.io/kcl-lang/k8s:1.29'
downloading 'kcl-lang/k8s:1.28' from 'ghcr.io/kcl-lang/k8s:1.28'
downloading 'kcl-lang/k8s:1.29' from 'ghcr.io/kcl-lang/k8s:1.29'
add dependency 'k8s' successfully
```

Case 3:

Instead of upgrading all modules, cautious developers typically want to upgrade only one module, with as few other changes to the build list as possible. There is no support for this. Also no current support for downgrading dependency.

**PROPOSAL**

We will follow the MVS approach used in go package manager given the fact that the underlying strategy achieves reproducible builds without the need of a lock file.  \
 \
The Go package manager adopts a Minimum Version Selection (MVS) approach to determine which packages to include in the final list for building. MVS aims to create builds that closely mirror the dependencies used by the package author during development. This means that when a user builds a project, the dependencies chosen are as similar as possible to the ones the original author developed against.

Minimal Version Selection (MVS) operates on the assumption that each module specifies only the minimum versions of its dependencies, adhering to the import compatibility rule where newer versions are expected to be compatible with older ones. This means dependency requirements include only minimum versions, without specifying maximum versions or incompatible later versions.

version selection strategy is meant to provide algorithms for four operations on build list:

- **Construct the current build list:**

	The rough build list for package M would be just the list of all modules reachable in the requirement graph starting at M and following arrows. This can be accomplished through a straightforward recursive traversal of the graph, ensuring to skip nodes that have already been visited. The rough built list can then be converted to the final build list.

- **Upgrade all modules to their latest versions:**
		
	This can be achieved by running go get -u which will upgrade all the modules to their latest versions.

	Upgrading the modules would mean all arrows in the dependency graph is now pointing to the latest version of the modules. This will result in a upgraded dependency graph but changes in the dependency graph alone won't cause future builds to use the updated modules. To achieve this we need a change in our built list in a way that won't affect dependent packages built list, as upgrades should be limited to our package alone.

	At first glance, it would seem intutive to include all the updated packages in our built list. But, not all packages are necessary and we want to include as few additional modules as possible. To produce a minimum requirement list, an helper algorithm R is introduced.

	**Algorithm R:**

	To compute a minimal requirement list inducing a given build list below the target, reverse postorder traversal is employed, ensuring modules are visited after all those pointing into them. Each module is added only if it's not implied by previously visited ones.

- **Upgrade one module to a specific newer version:**

	Upgrading all modules to their latest versions can be risky, so developers often opt to upgrade only one module.

	Upgrading one module mean that the arrow which earlier pointed to that module is now pointing to the upgraded version. We can construct a built list from the updated dependency graph, which can then be fed to Algorithm R to get a minimum requirement list.

- **Downgrade one module to a specific older version.**

	The downgrade algorithm examines each of the target's requirements separately. If a requirement conflicts with the proposed downgrade, meaning its build list contains a version of a module that is no longer allowed, the algorithm iterates through older versions until finding one that aligns with the downgrade.

	Downgrades make changes to the built list by removing requirements.

**Implementation**

The already implemented mvs library in go codebase can be reused with few modifications. \
https://github.com/golang/go/tree/master/src/cmd/go/internal/mvs
