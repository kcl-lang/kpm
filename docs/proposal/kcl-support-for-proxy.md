### Enhancement Proposal:  Proxy Support in KPM Downloader`

**Author:** [SkySingh04](https://linktr.ee/skysingh04)

**Objective**
The goal of this document is to propose a solution for implementing proxy support in the KPM downloader module, inspired by the proxy setup capabilities in Go Modules and Cargo. This feature aims to improve the stability and speed of downloading third-party dependencies, especially in environments constrained by network or firewall restrictions.

---

### Problem Statement
Currently, the KPM downloader faces challenges with slow or unstable downloads due to:
1. Network limitations, such as bandwidth throttling or high latency.
2. Environments requiring proxy configurations, such as corporate firewalls or restrictive ISPs.

To address these issues, introducing a proxy configuration option in the KPM downloader will provide flexibility and enhance usability across diverse environments.

---

### Solution Overview
The proposed solution involves:
1. Adding a `ProxyURL` configuration field to the downloader settings.
2. Utilizing custom HTTP clients that support proxy configurations.
3. Ensuring seamless proxy support for both Git and OCI (Open Container Initiative) downloads.
4. Providing an intuitive interface for users to enable and configure proxies.

---

### Key Features of the Solution

#### 1. ProxyURL Configuration
- Introduce a new configuration parameter, `ProxyURL`, in the KPM configuration file (e.g., `.kpmconfig`).
- Allow users to specify the proxy URL in the format:
  ```
  ProxyURL = "http://username:password@proxy.example.com:8080"
  ```
- Support both HTTP and HTTPS proxies.

#### 2. Proxy Setup in HTTP Client
- Modify the HTTP client used by KPM to utilize the `ProxyURL` field if specified.
- Use Go’s `http.Transport` to set the proxy configuration dynamically:
  ```go
  proxyURL, err := url.Parse(config.ProxyURL)
  if err != nil {
      log.Fatalf("Invalid proxy URL: %v", err)
  }
  transport := &http.Transport{
      Proxy: http.ProxyURL(proxyURL),
  }
  client := &http.Client{Transport: transport}
  ```

#### 3. Proxy Support for Git Operations
- Enhance Git download commands to respect the `ProxyURL` configuration by setting the `http.proxy` Git config property:
  ```bash
  git config --global http.proxy http://username:password@proxy.example.com:8080
  ```
- Ensure proxy settings are applied only when `ProxyURL` is specified in the configuration.

#### 4. Proxy Support for OCI Downloads
- Modify OCI registry client setup to utilize the custom HTTP client configured with the proxy settings.
- Example in Go:
  ```go
  resolver := docker.NewResolver(docker.ResolverOptions{
      Hosts: config.DockerHosts,
      Client: client, 
  })
  ```

#### 5. Fallback and Error Handling
- If the `ProxyURL` is invalid or unreachable, provide clear error messages to the user.
- Implement a fallback mechanism to bypass the proxy and attempt direct downloads when possible.

---

### Comparison with Go Mod and Cargo

#### Go Mod
- Supports setting `GOPROXY` for module downloads and `HTTP_PROXY`/`HTTPS_PROXY` environment variables for HTTP requests.
- Proposed KPM solution aligns with this by offering a `ProxyURL` configuration and leveraging the environment variables as a fallback.

#### Cargo
- Allows specifying proxy configurations in `.cargo/config`.
- Proposed solution mirrors this by introducing proxy support in the KPM configuration file and ensuring compatibility across different dependency sources.

---

### Migration Strategy
1. **Backward Compatibility**: Ensure existing KPM setups continue to work without proxy configurations.
2. **Incremental Rollout**: Release the proxy feature in phases:
   - Phase 1: Basic proxy support for HTTP/HTTPS downloads.
   - Phase 2: Comprehensive support for Git and OCI downloads.
3. **User Feedback**: Gather feedback from early adopters to refine the implementation.

---

### Risks and Mitigations
- **Invalid Proxy Configuration**: Validate the `ProxyURL` format during configuration parsing to prevent runtime errors.
- **Performance Overhead**: Test the impact of proxy usage on download speed and optimize as necessary.
- **Security Concerns**: Ensure sensitive information (e.g., proxy credentials) is handled securely and not logged.

---

### Implementation Plan
1. **Research**:
   - Study the proxy implementations in Go Mod and Cargo.
   - Identify libraries or utilities to simplify proxy handling in KPM.
2. **Development**:
   - Add `ProxyURL` field to the configuration structs.
   - Update HTTP, Git, and OCI clients to support proxy settings.
3. **Testing**:
   - Create test cases for various proxy configurations.
   - Simulate different network environments to validate the solution.
4. **Documentation**:
   - Update the KPM user guide with proxy setup instructions.

---

### Conclusion
Implementing proxy support in KPM will enhance its usability in restricted network environments and align it with industry standards set by Go Mod and Cargo. This solution provides a robust foundation for stable and efficient dependency downloads, ensuring KPM remains a competitive and user-friendly package manager.

