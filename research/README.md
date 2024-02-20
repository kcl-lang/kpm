# [WIP] Version management strategies

## Introduction
We have to propose a version management feature for KCL package manager.

## Features

1. View: The view feature allows users to see detailed information about modules and their dependencies, such as version numbers, dependencies, and metadata. This feature helps developers understand the structure of their project's dependencies and identify potential issues. 

2. Install: The install command enables users to download and install dependencies for a project. This feature automatically resolves dependencies and ensures that the correct versions are installed to prevent conflicts. 

3. Replacement: Allows substituting a module with another one, altering the module graph to accommodate different dependencies. 

4. Exclusion: Removes specific versions of a module from consideration during dependency resolution. It's specified in the go.mod file of the main module, redirecting requirements to the next higher version.

5. Upgrades: Updates modules to newer versions, potentially adding or removing indirect dependencies.

6. Downgrades: Reverts modules to previous versions, potentially removing higher versions and their dependent modules. 

## Content


## Summary
