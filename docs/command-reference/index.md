# kpm

The kpm cli

## Usage

```shell
kpm  <command> [arguments]...
```

## Description

`kpm` is a kcl package manager. It is used to install, remove, and update kcl packages.

## Options

### --help, -h

Show help for kpm command

### --version, -v

Print the version of kpm

## Subcommands

- [kpm init](./1.init.md) - Init a kcl package
- [kpm add](./2.add.md) - Add a dependency to a kcl package
- [kpm pkg](./3.pkg.md) - Package a kcl package into `*.tar``
- [kpm metadata](./4.metadata.md) - Print the metadata of a kcl package
- [kpm run](./5.run.md) - Compile a kcl package into yaml
- [kpm login](./6.login.md) - Login to a kcl registry
- [kpm logout](./7.logout.md) - Logout from a kcl registry
- [kpm push](./8.push.md) - Push a kcl package to a registry
- [kpm pull](./9.pull.md) - Pull a kcl package from a registry
- [kpm help](./10.help.md) - print help for kpm command
