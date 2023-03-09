# [WIP] How does kpm work.

**Note: this is an idea under discussion, not the final version.**

## Init an empty kcl module

In an empty directory, use kpm to initialize an empty kcl module.

```
$ kpm init my_kcl
```

kpm will generate a kcl.mod file and fill it with some default information.

```
[package]
name = "my_kcl"
version = "0.0.1"
edition = "0.4.5"
```

## Add dependencies

Add dependencies via kpm.

```
$ kpm add -git https://github.com/zong-zhe/konfig.git
$ kpm add -git https://github.com/zong-zhe/konfig1.git@v0.0.1
$ kpm add -git https://github.com/zong-zhe/konfig2.git@dshf894h9
$ kpm add -git https://github.com/zong-zhe/konfig3.git@main
```

Kpm will save all downloaded packages in a flat way in a directory.

The directory is `~/.kpm` by default or is specified by the environment variable `KPM_HOME`.

`kpm add` will add dependencies to the `kcl.mod` as below

```toml
[dependencies]
# Default is branch "main"
konfig = {git = "<github url>"} 
konfig1 = {git = "<github url>", version = "0.0.1"} 
konfig2 = {git = "<github url>", commit = "dshf894h9"} 
konfig3 = {git = "<github url>", branch = "main"} 
```

And kpm generates `kcl.mod.lock`. 

```
[[package]]
name = "konfig"
version = "0.0.1"
source = "git+https://github.com/xxxxx"
checksum = "d468802bab17cbc0cc575e9b0"
......
```

## Import dependencies

The import algorithm of kcl for absolute paths is as follows:

- The semantics of import a.b.c.d is

  - Search the path ./a/b/c/d from the current directory.
  - The current directory search fails, search from the root path ROOT_PATH/a/b/c/d.
 
- The definition of the root path ROOT_PATH is 
  - Look up the directory corresponding to the kcl.mod file from the current directory.
  - If kcl.mod is not found, read from the environment variable KCL_MODULE_ROOT e.g., kclvm/lib/*.

It could be either a subpath `./a/b/c/d` under the ROOT_PATH or an external package `a`, if you use import statement as follows:

```
import a.b.c.d
```

The import does not distinguish whether the module is internal or external. Therefore, if there is a ./a/b/c/d under the ROOT_PATH, it is considered that the current kcl module depends on an internal module named a, and an error will be raised when importing an external module with the same name a.

## Compile KCL module

After importing the externally dependent kcl module, the command to compile kcl needs to add an argument describing the external module selected for compilation.

```
kcl --extern konfig="<path>" --extern konfig1="<path>" main.k
```

The direct dependencies `konfig`, `konfig1` and dependencies path are passed to kclvm through `--extern`.

That's a lot of work, when a module depends on a lot of external modules, it's not really appropriate to manually type a full build command, so after `kpm init`, `kpm add` the `kpm download`, the next is `kpm build`.

Run `kpm build main.k` in a directory containing kcl.mod, and kpm will call kclvm based on the contents of kcl.mod.

```
$ kpm build main.k

$ kcl --extern konfig="<path>" --extern konfig1="<path>" main.k
```

If two dependencies `konfig` and `konfig1` are described in kcl.mod, then these two commands are equivalent.



