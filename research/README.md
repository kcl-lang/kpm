# [WIP] Version management strategies

## Abstract
This documentation presents the design and implementation of a version management system tailored for managing dependencies in KCL project. The system encompasses functionalities such as viewing detailed module information, installing dependencies, handling replacements and exclusions, as well as performing upgrades and downgrades. Each functionality is discussed in detail along with example usage scenarios and implementation strategies.

## Introduction
Version management plays a critical role in software development, especially when dealing with complex dependency structures. Managing dependencies effectively ensures project stability, compatibility, and ease of maintenance. In this documentation, I have proposed a comprehensive version management system equipped with various functionalities to address the challenges of dependency management.

## Operations on Build Lists

1. View:
The view functionality allows users to retrieve detailed information about modules and their dependencies. This feature aids developers in understanding the structure of their project's dependencies and identifying potential issues. Information such as version numbers, dependencies, and metadata can be queried.

2. Install:
The install command enables users to download and install dependencies for a project. It automatically resolves dependencies and ensures that the correct versions are installed to prevent conflicts. This functionality streamlines the setup process for new projects and ensures consistent development environments.

3. Replacement:
The replacement functionality allows substituting a module with another one, altering the module graph to accommodate different dependencies. This feature is useful when a particular module needs to be replaced with a compatible alternative without affecting other dependencies.

4. Exclusion:
Exclusion functionality removes specific versions of a module from consideration during dependency resolution, redirecting requirements to the next higher version. This feature enables users to exclude problematic versions or temporarily bypass certain dependencies.

5. Upgrades:
The upgrade functionality updates modules to newer versions, potentially adding or removing indirect dependencies. This feature ensures that projects stay up-to-date with the latest enhancements and bug fixes while maintaining compatibility with existing dependencies.

6. Downgrades:
The downgrade functionality reverts modules to previous versions, potentially removing higher versions and their dependent modules. This feature is valuable when encountering issues with newer versions or when reverting to a known stable state.

## Implementation

The version management system is implemented as a command-line tool, providing users with a familiar interface for interacting with dependencies. The system is designed to be modular and extensible, allowing for easy integration into existing development workflows.

Example usage scenario demonstrating the functionalities of the version management system:

 ```bash
$ kpm view <module_name>
```
 ```bash
$ kpm install
```
 ```bash
$ kpm replace <old_module> <new_module>
```
 ```bash
$ kpm exclude <module> <version>
```
 ```bash
$ kpm upgrade <module>
```
 ```bash
$ kpm downgrade <module> <version>
```

## Content

Survey and technology selection

### Semantic Versioning in NPM

The NPM package library is like a huge store with millions of items used by almost every JavaScript program out there. To keep things organized and safe, they use a system called "semantic versioning" (semver), which helps to update programs without breaking them. 

This research focuses on understanding how developers use a system called semantic versioning in the NPM ecosystem, which is like a big library of code used by many programs. and want to see how this affects the security of the software supply chain. Here are the main questions we're trying to answer:

RQ1: Do developers set rules for how their code should be updated automatically?
RQ2: Do developers follow the rules of semantic versioning to make sure updates happen smoothly for other programs using their code?
RQ3: Are there many cases where parts of programs are outdated? And when updates are made available, how long does it take for other programs to get them?
RQ4: What kinds of changes do developers usually make when they update their code? And how often do they just update the parts that their code depends on?

These questions help us understand how software updates work in this big library of code and how it affects the safety and reliability of the software we use.

### RQ1: Version Constraint Usage

We have to understand how developers tell their software which versions of other software it can use. In the NPM system, developers have different ways to set these rules, but not sure which ones they use most often and how strict or flexible those rules are.

 grouped these rules into different categories:

1. Exact rules: These only allow one specific version of the software.
2. Bug-flexible rules: These allow updates that fix bugs, but nothing else major.
3. Minor-flexible rules: These allow updates that add new features, but nothing that breaks things.
4. Greater than or equal rules: These allow any version that's the same or newer than a specific version.
5. Any rules: These allow any version at all.
6. Other rules: These are less common and can include special cases like using web links to specify versions.

### RQ2: Semantic Versioning in Updates

We have to investigate how developers update their software versions in the NPM ecosystem using semantic versioning. Semantic versioning means developers label their updates as bug fixes, minor changes (like adding features), or major changes (which could break things). 

In NPM, versions can be published in a non-chronological order, which means developers can maintain different branches of their software. For example, a developer might release version 1.0.0, then version 2.0.0, and then go back to release version 1.0.1. We need to understand these updates in the right order to analyze them correctly.

To do this,  grouping versions by their major component and make sure they're in chronological order within each group. Then we look at updates between versions in the same group and between different groups. focus on updates that aren't pre-releases (like beta versions) and make sure the updates follow a consistent order.

Once  all the updates sorted out,  look at how often developers release bug fixes, minor changes, and major updates. also look at updates that fix or introduce vulnerabilities. This helps us understand how developers manage their software updates and how they handle security issues.

### RQ3: Out-of-Date Dependencies and Update Flows

Now looking at how often NPM packages are behind on updates and how long it takes for updates to reach other packages that depend on them. This involves considering all the packages a particular package relies on. However, figuring this out accurately is tricky because NPM's system for resolving dependencies is complicated, and  need to be done for different points in time.

To tackle this, developer's have come up with a way to accurately solve dependencies and see how things change over time. and use a combination of NPM's solver and a tool that helps us understand what the world of NPM looked like at different times in the past.

With this setup,  conduct two experiments: first,  look at the latest version of every NPM package to see how many are using outdated dependencies. Then,  track how updates spread to other packages by checking when a dependent package gets the latest updates over time.

### RQ4: Analyzing Code Changes in Updates

After studying how developers label updates for software packages,  see what actually changes when these updates happen. So,the code from the old version is to be examined and the new version of each package and compared them. We looked at things like if they changed any other software that the package relies on (like tools or libraries), if they changed the actual code files (like JavaScript files), or if they changed both.

Then,  sort these changes based on how big the update was. For example, was it a big change, a small one, or just a fix for a problem? This way, we could see if different types of updates tended to change certain things in the code more often.

It's worth noting that sometimes it's tricky to see exactly what changed, especially if the code is "compiled" or "minified," which means it's been compressed to make it run faster. Also,  focusing specifically on changes in JavaScript code and didn't include other types of files, like scripts written in other languages.


## System Architecture

A special system were made up of three main parts:

1. Metadata Manager: This part of the system keeps track of all the information about the packages one is interested in. It constantly watches for any changes in the package data and stores it in a neat and organized way. Think of it as a librarian who keeps track of all the books in a library.

2. Job Manager: When the Metadata Manager finds something interesting, like a new version of a package, it tells the Job Manager to do something about it. The Job Manager then figures out what needs to be done and assigns tasks to different parts of our system to handle them.

3. Compute Cluster: This is like the engine of our system. It's where all the heavy lifting happens. It's made up of many computers working together, like a team, to download and analyze the code in these packages.

Here's how each task is tackled:

A. Getting Package Information:  collecting information about these packages.  focusing on a database called NPM, which holds a lot of data about JavaScript packages. However, the data there is a bit messy, so  cleaning it up and stored it in a more organized way using a tool called PostgreSQL.

B. Downloading and Storing Package Code: store all the code from these packages to analyze it later.  a special storage system was built that can handle lots of files at once. When we need to download code from a package, the Job Manager assigns a computer in Compute Cluster to do it.

C. Looking Back in Time: To understand how packages used to work,  a special tool was created that lets us travel back in time to see how things were in the past. It's like looking at old snapshots of a package to see what it used to be like.

By using this system, we were able to study how updates to JavaScript packages work and how they affect other parts of the code. It's like taking apart a complicated machine to see how each part works and how they all fit together.

### Advantages and Disadvantages

Advantages of Semantic Versioning:

1. Clear Communication: Semantic versioning provides a standardized way for developers to communicate the nature of changes in their software. By using a clear numbering system (major.minor.patch), developers can quickly understand the significance of an update.

2. Predictable Updates: With semantic versioning, developers know what to expect when they update a package. They can anticipate whether an update is likely to introduce new features, improvements, or just bug fixes, helping them manage their software dependencies more effectively.

3. Dependency Management: Semantic versioning facilitates dependency management by allowing developers to specify version constraints. This helps ensure that their software works correctly with compatible versions of dependent packages, reducing compatibility issues.

4. Risk Mitigation: By distinguishing between major, minor, and patch updates, semantic versioning helps mitigate the risk of introducing breaking changes. Developers can choose to adopt updates selectively based on their impact, reducing the likelihood of unexpected issues.

Disadvantages of Semantic Versioning:

1. Complexity: While semantic versioning provides a clear framework for versioning, it can still be complex to manage dependencies, especially in larger projects with many interconnected packages. Ensuring compatibility across different versions can require careful coordination and testing.

2. Limited Scope: Semantic versioning primarily addresses changes in software functionality, but it may not fully capture other aspects such as security vulnerabilities or performance improvements. Developers may need additional mechanisms to track and address these concerns effectively.

3. Dependency Lock-In: Strict version constraints based on semantic versioning can sometimes lead to dependency lock-in, where developers are hesitant to update dependencies due to concerns about compatibility. This can result in using outdated or insecure versions of software.

4. Interpretation Variability: Despite the guidelines provided by semantic versioning, there can still be variability in how developers interpret and apply version numbers. Differences in interpretation may lead to inconsistencies or misunderstandings when managing dependencies.

### Minimal Version Selection Go & Versioning

Imagine you're building a complex structure out of Lego pieces. Each piece depends on others to fit together properly. Now, imagine you have a list of specific Lego pieces you need to use to build something. This list is like your "build list."

Here's how the Lego version selection works:

1. Constructing the current build list: You start with the pieces you need for your current project. Then, you add all the pieces those pieces depend on. But if a certain type of piece is needed multiple times, you only grab the newest version of it.

2. Upgrading all pieces to their latest versions: You go through your list and pick the latest version of each type of piece. This ensures you're using the newest available parts.

3. Upgrading one piece to a specific newer version: You first build your structure with the current pieces. Then, if you want to use a newer version of a specific piece, you just swap that piece out with the newer version. But remember, you still need to make sure everything fits together with the new piece.

4. Downgrading one piece to a specific older version: If you want to use an older version of a piece, you rewind to the point where you were using the older version. You then make sure all the other pieces in your list still fit with this older version.

These operations ensure that your Lego structure is built with the right pieces, in the right versions, and that everything fits together smoothly. It's simple, efficient, and easy to manage.

### Low-Fidelity Builds

The way Go currently picks which versions of modules to use isn't great. It has two main methods, but both can lead to problems.

1. The first method is like going to a store and picking out the latest version of each item on your shopping list. But sometimes, it might grab an old version of something you already have at home, even if there's a newer one available. This can mess up your project because you end up with outdated parts.

2. The second method is like going to the store and getting all the newest versions of everything, even if they're not exactly what you need. This can cause issues because you might end up with parts that haven't been tested together or might not fit well with the rest of your project.

Both of these methods result in what we call "low-fidelity builds." It means the final project you build isn't exactly like what the original creator built. This can be frustrating because it introduces unnecessary differences without any good reason.

### Theory
- Minimal version selection simplifies version selection by sticking strictly to the versions specified in requirements, which avoids the complexity of solving general Boolean satisfiability problems.
- It falls within the intersection of three tractable SAT subproblems: 2-SAT, Horn-SAT, and Dual-Horn-SAT, ensuring efficient algorithms for version selection.
- The uniqueness and minimality properties of minimal version selection make it simple and efficient to implement, ensuring high-fidelity builds.

Excluding Modules:
- Module exclusions must be unconditional to maintain uniqueness and minimality properties.
- Exclusions are implemented as unconditional constraints, ensuring they are decided independently of build selections.

Replacing Modules:
- Modules can declare replacements to replace a specific version with another.
- Replacements are implemented by modifying the module requirement graph, maintaining simplicity and efficiency.

Who Controls Your Build?:
- Module authors have control over their own builds but not over other users' builds.
- Balancing control between dependencies and top-level modules ensures predictability and avoids conflicts.

High-Fidelity Builds:
- Minimal version selection provides high-fidelity builds by using the oldest version that meets requirements, ensuring reproducibility.
- Unlike other systems that use the newest version available, minimal version selection prioritizes the version the author used, avoiding unnecessary deviations.

Upgrade Speed:
- Minimal version selection ensures that dependencies move forward at an appropriate pace, avoiding unnecessary upgrades.

Upgrade Timing:
- Upgrades in minimal version selection occur only when explicitly requested by the developer, ensuring control over changes to the build.

Minimality:
- Minimal version selection is designed to be minimal and understandable, prioritizing simplicity and predictability over raw flexibility.
- It aims to provide a version selection algorithm that is predictable, boring, and easy to understand.

## Summary

Overall, while semantic versioning and Minimal Version Selection offers significant benefits in terms of clarity and predictability, it is important for developers to be mindful of its limitations and to supplement all priorily present version managers with additional practices for effective software management.