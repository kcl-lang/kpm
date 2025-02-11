# Analysis of Cargo Test Architecture

**Author**: [Siddhi Agrawal](https://github.com/Siddhi-agg)

### Note: The code snippets have been picked directly from the `Cargo test` tool and have been shortened at some areas of the document to maintain readability.

## 1. Cargo Test Macro (Entry point)

The `cargo_test` macro is a core testing point that provides test isolation and conditional execution. It is defined in the `cargo-test-macro` crate. It sets up a new and isolated filesystem with mock home directories to ensure the test runs cleanly and the user's actual environment isn't disturbed. The  registries are also mocked to simulate a package registry.

This can handle test execution defined by some required conditions like a specific rust version, containers and network access.

```rust
#[proc_macro_attribute]
pub fn cargo_test(attr: TokenStream, item: TokenStream) -> TokenStream {
    let mut ignore = false;
    let mut requires_reason = false;
    let mut explicit_reason = None;
    let mut implicit_reasons = Vec::new();

    // Parse test attributes/conditions
    for rule in split_rules(attr) {
        match rule.as_str() {
            "nightly" => {
                if is_not_nightly {
                    set_ignore!("only works on nightly");
                }
            }
            s if s.starts_with(">=") => {
                let version = &s[2..];
                if !meets_version_requirement(version) {
                    set_ignore!(format!("requires rustc {}", version));
                }
            }
            // ...other conditions
        }
    }

    // Generate test function with setup
    let test_fn = format!(
        "#[test]
        fn {name}() {{
            cargo_test_support::paths::init_root();
            let _g = cargo_test_support::paths::CargoPathGuard::new();
            {body}
        }}"
    );
} 
```

## 2. Project Builder

The `ProjectBuilder` implements an API for creating test projects with a natural syntax. It manages file creation, directory/folder structures, and project configuration.

It allows us to create projects to run the tests against. It helps create the required folder structure and then write files to it. It also records if the project contains a `cargo.toml` file or not, and adds it to the project if it is initially not present. As required by the tests, we can also create symlinks to files and directories.  

 The `Project` implementation also provides us the functionality to build a test project directly from a fixed template. We can perform various functions fetching the path of various key files and directories of the project, reading and writing to files and executing a program using `ProcessBuilder`. ` ProcessBuilder` can also be used to run cargo commands on files. Cargo test supports custom test runners via `Execs` and `ProcessBuilder`, enabling flexible CLI interactions.

```rust
pub struct ProjectBuilder {
    root: Project,
    files: Vec<FileBuilder>,
    symlinks: Vec<SymlinkBuilder>,
    no_manifest: bool,
}

impl ProjectBuilder {
    /// Adds a file to the project.
    pub fn file<B: AsRef<Path>>(mut self, path: B, body: &str) -> Self {
        self._file(path.as_ref(), body, false);
        self
    }

    /// Creates the project.
    pub fn build(mut self) -> Project {
        // First, clean the directory if it already exists
        self.rm_root();

        // Create the empty directory
        self.root.root().mkdir_p();

        //Add a manifest if not already present
        let manifest_path = self.root.root().join("Cargo.toml");
        if !self.no_manifest && self.files.iter().all(|fb| fb.path != manifest_path) {
            self._file(
                Path::new("Cargo.toml"),
                &basic_manifest("foo", "0.0.1"),
                false,
            )
        }

        let past = time::SystemTime::now() - Duration::new(1, 0);
        let ftime = filetime::FileTime::from_system_time(past);

        for file in self.files.iter_mut() {
            file.mk();
            if is_coarse_mtime() {
                filetime::set_file_times(&file.path, ftime, ftime).unwrap();
            }
        }

        for symlink in self.symlinks.iter_mut() {
            symlink.mk();
        }

        let ProjectBuilder { root, .. } = self;
        root
    }
}
```

## 3. Network Mocking Architecture

### **Core Architecture**

The network mocking system in Cargo is built around the `TestRegistry` and `RegistryBuilder` structs, which provide a network simulation of a package registry. `TestRegistry` represents a local and live instance of a package registry which can be interacted with during tests. `RegistryBuilder` can be used to configure the local registry as per the needs. 

`Package` struct is a simulation of crate packages and they are configured to be used instead of actual crate.io packages.

```rust
pub struct TestRegistry {
    server: Option<HttpServerHandle>,
    index_url: Url,
    path: PathBuf,
    api_url: Url,
    dl_url: Url,
    token: Token,
}

pub struct RegistryBuilder {
    alternative: Option<String>,
    token: Option<Token>,
    auth_required: bool,
    http_index: bool,
    http_api: bool,
    api: bool,
    configure_token: bool,
    configure_registry: bool,
    custom_responders: HashMap<String, RequestCallback>,
    not_found_handler: RequestCallback,
    delayed_index_update: usize,
    credential_provider: Option<String>,
}

pub struct Package {
    name: String,
    vers: String,
    deps: Vec<Dependency>,
    files: Vec<PackageFile>,
    yanked: bool,
    features: FeatureMap,
    local: bool,
    alternative: bool,
    invalid_index_line: bool,
    index_line: Option<String>,
    edition: Option<String>,
    resolver: Option<String>,
    proc_macro: bool,
    links: Option<String>,
    rust_version: Option<String>,
    cargo_features: Vec<String>,
    v: Option<u32>,
}
```

### **Index Management**

The system maintains a Git-based indexing that is very similar to crates.io's structure. The index is created based on the version information and other metadata of the new crate.

```rust
fn save_new_crate(
    dst: PathBuf,
    new_crate: crates_io::NewCrate,
    file: &[u8],
    file_cksum: String,
    registry_path: &Path,
) {
    //... Write the package `.crate`
    
    //... Configure the dependencies of the new crate

    //... Write the index line
    let line = create_index_line(
        serde_json::json!(new_crate.name),
        &new_crate.vers,
        deps,
        &file_cksum,
        new_crate.features,
        false,
        new_crate.links,
        new_crate.rust_version.as_deref(),
        None,
    );
    write_to_index(registry_path, &new_crate.name, line, false);
}
```

### **HTTP API Simulation**

The HTTP API simulation provides mock endpoints like crates.io's API. This includes:

- **Publishing Endpoint** (`/api/v1/crates/new`): While processing package uploads, it validates the package's metadata and its checksum, and then validates the token against cryptographic key. If all checks pass, a new crate is published, otherwise an appropriate response is generated.

- **Download Endpoint** : While processing package downloads, it validates the token against cryptographic key. If the validation passes, the crate is downloaded from the registry. 

```rust
impl HttpServer {
    fn route(&self, req: &Request) -> Response {
        match (req.method.as_str(), path.as_slice()) {
            // Package downloads
            ("get", ["dl", ..]) => {
                if !self.check_authorized(req, None) {
                    self.unauthorized(req)
                } else {
                    self.dl(&req)
                }
            },
            // Package publishing
            ("put", ["api", "v1", "crates", "new"]) => self.check_authorized_publish(req),
            // Yank operations
            ("delete" | "put", ["api", "v1", "crates", name, version, mutation]) => {
                // Handle yank/unyank
            },
            // Owner management
            ("get" | "put" | "delete", ["api", "v1", "crates", name, "owners"]) => {
                // Handle owner operations
            }
        }
    }
}
```

### **Authentication System**
The authentication system uses PASETO (Platform-Agnostic Security Tokens) tokens for secure registry access. Initially, the token type is checked and then PASETO validation logic is performed. Time-based validation is also applied. For the operations that modify the package like yank/unyank and publish, extra checks are applied.

```rust
impl HttpServer {
    fn check_authorized(&self, req: &Request, mutation: Option<Mutation<'_>>) -> bool {
        let (private_key, private_key_subject) = match &self.token {
            Token::Plaintext(token) => return Some(token) == req.authorization.as_ref(),
            Token::Keys(private_key, private_key_subject) => {
                (private_key.as_str(), private_key_subject)
            }
        };

        // Validate PASETO token
        let secret: AsymmetricSecretKey<pasetors::version3::V3> = private_key.try_into().unwrap();
        let public: AsymmetricPublicKey<pasetors::version3::V3> = (&secret).try_into().unwrap();
        // ... token validation logic
    }
}
```

### **Package Publishing**
The cargo test architecture implements a complete package publishing workflow through the HTTP API. When a package is published, authorization is validated using PASETO or plaintext tokens. Then a folder structure is created and the `.crate` file is appropriately placed. The index is updated using the new package information. 

During publishing, checksums are validated, version management is supported and dependencies are tracked.

```rust
impl HttpServer {
    pub fn check_authorized_publish(&self, req: &Request) -> Response {
        if let Some(body) = &req.body {
            //... Process package upload

            // Get the metadata of the package
            let (len, remaining) = body.split_at(4);
            let json_len = u32::from_le_bytes(len.try_into().unwrap());
            let (json, remaining) = remaining.split_at(json_len as usize);
            let new_crate = serde_json::from_slice::<crates_io::NewCrate>(json).unwrap();
            // Get the `.crate` file
            let (len, remaining) = remaining.split_at(4);
            let file_len = u32::from_le_bytes(len.try_into().unwrap());
            let (file, _remaining) = remaining.split_at(file_len as usize);
            let file_cksum = cksum(&file);

            if !self.check_authorized(
                req,
                Some(Mutation {
                    mutation: "publish",
                    name: Some(&new_crate.name),
                    vers: Some(&new_crate.vers),
                    cksum: Some(&file_cksum),
                }),
            ) {
                return self.unauthorized(req);
            }

            // Store package and update index
            save_new_crate(dst, new_crate, file, file_cksum, &self.registry_path);
        }
    }
}
```

This comprehensive network mocking system allows Cargo tests to verify network-dependent functionality without requiring actual network access, ensuring tests are reliable. Thes design has been made so that it is easy to extend it if more functionalities are needed while maintaining compatibility with the actual crates.io API.

## 4. Command Execution System

### **Architecture**
The command execution system or CLI interface in Cargo is built around the `Execs` struct, which provides a very detailed interface for running and verifying the execution of terminal commands. 

```rust
pub struct Execs {
    ran: bool,
    process_builder: Option<ProcessBuilder>,
    expect_stdin: Option<String>,
    expect_exit_code: Option<i32>,
    expect_stdout_data: Option<snapbox::Data>,
    expect_stderr_data: Option<snapbox::Data>,
    expect_stdout_contains: Vec<String>,
    expect_stderr_contains: Vec<String>,
    expect_stdout_not_contains: Vec<String>,
    expect_stderr_not_contains: Vec<String>,
    expect_stderr_with_without: Vec<(Vec<String>, Vec<String>)>,
    stream_output: bool,
    assert: snapbox::Assert,
}
```

### **Command Builder** 
The Cargo test system provides a reliable API for command building and its configuration. We can chain different methods to set different parts of the required configuration. 

The command builder supports environment variables addition and removal, working directory, and command-line arguments:

```rust
impl Execs {
    pub fn arg<T: AsRef<OsStr>>(&mut self, arg: T) -> &mut Self {
        if let Some(ref mut p) = self.process_builder {
            p.arg(arg);
        }
        self
    }

    pub fn env<T: AsRef<OsStr>>(&mut self, key: &str, val: T) -> &mut Self {
        if let Some(ref mut p) = self.process_builder {
            p.env(key, val);
        }
        self
    }
}
```

### **Environment Isolation**
The Cargo test tool ensures test isolation by setting up a clean environment for each and every test. Various environment variables are generally set before each test to ensure all required conditions are met.

```rust
pub trait TestEnvCommandExt: Sized {
    fn test_env(mut self) -> Self {
        // Clear cargo-specific configuration
        for (k, _v) in env::vars() {
            if k.starts_with("CARGO_") {
                self = self.env_remove(&k);
            }
        }
        //...Handle the RUSTUP_TOOLCHAIN environment variable and modify the PATH variable 
        
        self = self
            .current_dir(&paths::root())
            .env("HOME", paths::home())
            .env("CARGO_HOME", paths::cargo_home())
            // Many more environment variables...
            .env_remove("RUSTFLAGS");
        self
    }
}
```

### **Output Validation**
The output validation system of crago test is very advanced, and allows many types of output checking:

- Exact match validation
- Contains exit code validation
- Pattern matching
- Negative matching (does not contain)

```rust
impl Execs {
    pub fn with_stdout_contains<S: ToString>(&mut self, expected: S) -> &mut Self {
        self.expect_stdout_contains.push(expected.to_string());
        self
    }

    fn match_output(&self, code: Option<i32>, stdout: &[u8], stderr: &[u8]) -> Result<()> {
        // Validate exit code
        if let Some(expected) = self.expect_exit_code {
            if code != Some(expected) {
                bail!("process exited with code {} (expected {})", code.unwrap_or(-1), expected);
            }
        }

        // Validate output patterns
        for expect in self.expect_stdout_contains.iter() {
            compare::match_contains(expect, stdout, self.assert.redactions())?;
        }
        // ... more validation
        Ok(())
    }
}      
```
### **Process Execution Safety**
The architecture includes several safety features like Execution verification through Drop trait and Output validation requirements to prevent common testing mistakes being made.

### **Output Streaming**
Cargo test includes support for real-time output streaming during test execution, which can prove particularly very useful for debugging.

```rust
impl Execs {
    pub fn stream(&mut self) -> &mut Self {
        self.stream_output = true;
        self
    }

    fn match_process(&self, process: &ProcessBuilder) -> Result<RawOutput> {
        if self.stream_output {
            process.exec_with_streaming(
                &mut |out| {
                    println!("{}", out);
                    Ok(())
                },
                &mut |err| {
                    eprintln!("{}", err);
                    Ok(())
                },
                true,
            )
        } else {
            // Normal execution
        }
    }
}
```
This well-detailed command execution system provides a robust pillar for testing Cargo's CLI functionality, ensuring commands are executed correctly and their output is properly validated and test environments are clean and isolated.

## 5. Container Support
Cargo test supports Docker containers for running tests, whose lifecycle can be manipulated. That really helps in making port mapping easier.

```rust
pub struct Container {
    build_context: PathBuf,
    files: Vec<MkFile>,
}

pub struct ContainerHandle {
    name: String,
    ip_address: String,
    port_mappings: HashMap<u16, u16>,
}
```

This document provides an in-depth description at Cargo's test framework, covering network mocking, CLI interface and command execution, file-system isolation, and containerization in tests to ensure reliability in Rust projects.
