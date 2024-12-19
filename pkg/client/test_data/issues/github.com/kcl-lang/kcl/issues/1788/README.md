## Bug Report 

https://github.com/kcl-lang/kcl/issues/1788

When I run the kcl run path/to/files command from a directory that does not contain a root path, kcl.mod and kcl.mod.lock files are generated in the directory specified on the command line path to run. However, I did not intend for the specified path to become a root path and it can be unclear why it is being treated as one. Why does the kcl.mod file get created automatically? Is there any way to change this default behavior so that I don't have files auto generated in places I don't expect and that will affect the results of running kcl code?

### What is your KCL components version? (Required)

kcl version: 0.11.0-alpha.1
