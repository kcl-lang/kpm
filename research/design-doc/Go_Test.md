# **Designing a Robust Testing Framework for KCL**  
**Author**: [Rashid Alam](https://github.com/7h3-3mp7y-m4n)

**Note:** The code snippets are taken from a Go package, with some sections shortened and modified for assumption-based explanations.

## **Unit & End-to-End Testing with a Mock Runtime**  

### **1. Introduction**  
To enhance the stability of the KCL package management tool and make it more accessible for community developers, we propose a new testing framework. This framework will support both **unit testing** and **end-to-end (E2E) testing**, ensuring robust validation of KCL functionalities in a controlled environment.  

### **2. CLI Integration for Testing**  
We will introduce new subcommands within the default KCL CLI:  

- **Unit Testing:**  
  ```sh
  kcl test -U
  ```
- **End-to-End Testing:**  
  ```sh
  kcl test -e2e
  ```  

These commands will allow users to easily execute tests without additional configuration.

---

## **3. Implementation Using Golang**  
To implement this framework, we will leverage multiple Golang packages, including:  

### **3.1 CLI Management with Cobra**  
[Cobra](https://github.com/spf13/cobra) is a widely used package for building Go-based CLI applications. It provides a structured approach using **commands, arguments, and flags**, making it easy to create intuitive and extensible CLI tools.  

#### **Example: Defining a Root Command**  
```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kcl",
	Short: "KCL is a powerful package management tool",
	Long: `KCL provides a robust package management system 
with support for unit and end-to-end testing.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Execute default command logic
		fmt.Println("KCL CLI executed")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
```
### **3.2 Using Flags with the Standard Library**

If we prefer to use Go’s standard library instead of Cobra, we can leverage the built-in `flag` package:

```go
/ These examples demonstrate more intricate uses of the flag package.
package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

// Example 1: A single string flag called "species" with default value "gopher".
var species = flag.String("species", "gopher", "the species we are studying")

// Example 2: Two flags sharing a variable, so we can have a shorthand.
// The order of initialization is undefined, so make sure both use the
// same default value. They must be set up with an init function.
var gopherType string

func init() {
	const (
		defaultGopher = "pocket"
		usage         = "the variety of gopher"
	)
	flag.StringVar(&gopherType, "gopher_type", defaultGopher, usage)
	flag.StringVar(&gopherType, "g", defaultGopher, usage+" (shorthand)")
}

// Example 3: A user-defined flag type, a slice of durations.
type interval []time.Duration

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (i *interval) String() string {
	return fmt.Sprint(*i)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (i *interval) Set(value string) error {
	// If we wanted to allow the flag to be set multiple times,
	// accumulating values, we would delete this if statement.
	// That would permit usages such as
	//	-deltaT 10s -deltaT 15s
	// and other combinations.
	if len(*i) > 0 {
		return errors.New("interval flag already set")
	}
	for _, dt := range strings.Split(value, ",") {
		duration, err := time.ParseDuration(dt)
		if err != nil {
			return err
		}
		*i = append(*i, duration)
	}
	return nil
}

// Define a flag to accumulate durations. Because it has a special type,
// we need to use the Var function and therefore create the flag during
// init.

var intervalFlag interval

func init() {
	// Tie the command-line flag to the intervalFlag variable and
	// set a usage message.
	flag.Var(&intervalFlag, "deltaT", "comma-separated list of intervals to use between events")
}

func main() {
	// All the interesting pieces are with the variables declared above, but
	// to enable the flag package to see the flags defined there, one must
	// execute, typically at the start of main (not init!):
	//	flag.Parse()
	// We don't call it here because this code is a function called "Example"
	// that is part of the testing suite for the package, which has already
	// parsed the flags. When viewed at pkg.go.dev, however, the function is
	// renamed to "main" and it could be run as a standalone example.
}

```
---

## **4. Mock Runtime Environment for Offline Testing**  

### **4.1 Purpose**  
A **mock runtime** will enable offline testing by simulating network interactions (e.g., downloading and publishing packages) without requiring actual internet connectivity.

### **4.2 Functionality**  
- **Mock HTTP requests** for package registry interactions.  
- **Simulated filesystem operations** for storing package data, metadata, and logs.  
- **Offline independence** to ensure tests remain repeatable and reliable.  

### **4.3 Network Mocking Architecture**  
We will use the `httptest` package to simulate API interactions.

#### **Mock Endpoints:**  
| Method | Endpoint | Functionality |
|--------|---------|--------------|
| `GET` | `/dl/{package_name}` | Simulate downloading a package |
| `PUT` | `/api/v1/crates/new` | Simulate publishing a package |
| `DELETE` | `/api/v1/crates/{package_name}/yank` | Simulate package yank operation |
| `GET` | `/api/v1/crates/{package_name}/owners` | Fetch package owners |

#### **Example: HTTP Mock Server**  
```go
import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadPackage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "package": "test-package"}`))
	}))
	defer mockServer.Close()

	resp, err := http.Get(mockServer.URL + "/dl/test-package")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %v", resp.Status)
	}
}
```

---

## **5. Filesystem Mocking for Offline Testing**  

### **5.1 Purpose**  
Simulate filesystem operations **without affecting the actual file system** during tests. This will be useful for **storing package data, metadata, and logs** in an isolated environment.

### **5.2 Components**  
- **Temporary Files & Directories:** Use `os` and `ioutil` packages to create in-memory storage.  
- **Filesystem Simulations:** Mimic package storage and retrieval using Go’s `os` and `ioutil` libraries.

#### **Example: Temporary File Creation**  
```go
import (
	"io/ioutil"
	"os"
	"testing"
)

func TestTempFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "testpkg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up after test

	_, err = tmpFile.Write([]byte("test data"))
	if err != nil {
		t.Fatal(err)
	}
}
```

---

## **6. Unit and End-to-End Testing**  

### **6.1 Unit Testing**  
Unit tests will validate individual components such as:  
- **Package Validator**  
- **Metadata Parser**  
- **CLI Argument Processing**  

#### **Example: Unit Test for Metadata Parsing**  
```go
import (
	"testing"
)

func TestParseMetadata(t *testing.T) {
	expected := "test-package"
	got := ParseMetadata("test-package")
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}
```

### **6.2 End-to-End Testing**  
End-to-end tests will simulate the entire toolchain, covering:  
- **Downloading a package**  
- **Storing it in a mock filesystem**  
- **Publishing the package back to the registry**  

#### **Example: End-to-End Test**  
```go
func TestE2EFlow(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"package": "test-package"}`))
	}))
	defer mockServer.Close()

	resp, err := http.Get(mockServer.URL + "/dl/test-package")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %v", resp.Status)
	}
}
```

## Helm Test: Offline Testing Strategy

To meet the requirements for offline testing, we propose leveraging **Helm** in combination with local Helm repositories and mock Kubernetes environments. Helm is widely used for managing Kubernetes applications and can be adapted for offline testing using local chart repositories, local Kubernetes clusters, and mock servers for simulating external APIs.

### **1. Key Components and Architecture**

#### 1.1 **Local Helm Repositories**

-   Helm charts are typically pulled from remote repositories like `charts.helm.sh`. However, in an offline environment, charts will be stored in a **local directory**.
    
-   Helm provides a mechanism to add custom repositories, and by using the `helm repo index` command, we can simulate the behavior of remote repositories locally.
    
-   The Helm repository index is generated, pointing to a local directory containing packaged Helm charts.
    

#### 1.2 **Local Kubernetes Cluster**

-   **Minikube** or **Kind** (Kubernetes in Docker) will be used to create a local Kubernetes environment where Helm can deploy applications.
    
-   This local cluster mimics a real Kubernetes environment, allowing Helm to interact with it just as it would with a cloud-based cluster.
    

#### 1.3 **Mock External APIs**

-   External dependencies such as chart registries or other package-related APIs (e.g., token validation, metadata fetching) can be simulated using a **local mock server**.
    
-   A mock HTTP server will simulate the responses Helm would expect from an external chart registry, allowing for testing without network access.
    

----------

### **2. Solution Implementation**

#### 2.1 **Creating and Packaging Helm Charts**

-   Helm charts are created and packaged into `.tgz` files:
    
    ```
    helm create mychart
    helm package mychart
    ```
    

#### 2.2 **Setting Up a Local Helm Repository**

-   Create a directory to serve as the Helm chart repository:
    
    ```
    mkdir -p /tmp/helm-repo/charts
    ```
    
-   Move the packaged chart and generate an index file:
    
    ```
    mv mychart-1.0.0.tgz /tmp/helm-repo/charts
    helm repo index /tmp/helm-repo
    ```
    
-   Add the local Helm repository:
    
    ```
    helm repo add local-repo file:///tmp/helm-repo
    helm repo update
    ```
    

#### 2.3 **Setting Up a Local Kubernetes Cluster**

-   Start a Minikube cluster:
    
    ```
    minikube start --driver=docker
    ```
    
-   Verify the Kubernetes cluster:
    
    ```
    kubectl get nodes
    ```
    

#### 2.4 **Mocking External APIs**

-   Create a simple mock server in Go:
    
    ```
    package main
    
    import (
        "fmt"
        "net/http"
    )
    
    func chartHandler(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `{"name": "mychart", "version": "1.0.0"}`)
    }
    
    func main() {
        http.HandleFunc("/v1/charts/mychart", chartHandler)
        http.ListenAndServe(":8080", nil)
    }
    ```
    

#### 2.5 **Testing Offline with Helm**

-   Install, upgrade, and uninstall a Helm chart:
    
    ```
    helm install my-release local-repo/mychart
    helm upgrade my-release local-repo/mychart
    helm uninstall my-release
    ```
    

----------

### **3. Unit Testing in Isolation**

#### 3.1 **Mocking Network Dependencies**

-   Configure a local Helm repository to simulate chart downloads:
    
    ```
    mkdir -p /tmp/helm-repo/charts
    helm create mychart
    helm package mychart
    mv mychart-1.0.0.tgz /tmp/helm-repo/charts
    helm repo index /tmp/helm-repo
    helm repo add local-repo file:///tmp/helm-repo
    helm repo update
    ```
    

#### 3.2 **Mock HTTP Server for External APIs**

-   Example of a mock server handling chart metadata and token validation:
    
    ```go
    package main
    
    import (
        "fmt"
        "net/http"
    )
    
    func mockChartHandler(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"name": "mychart", "version": "1.0.0", "description": "A sample chart"}`)
    }
    
    func mockTokenValidation(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"valid": true}`)
    }
    
    func main() {
        http.HandleFunc("/v1/charts/mychart", mockChartHandler)
        http.HandleFunc("/api/v1/token/validate", mockTokenValidation)
        http.ListenAndServe(":8080", nil)
    }
    ```
    

#### 3.3 **Unit Test Example (Using Go’s Testing Framework)**

-   Example Go test to simulate Helm chart installation:
    
    ```go
    package package_test
    
    import (
        "testing"
        "os/exec"
        "github.com/stretchr/testify/assert"
    )
    
    func TestHelmChartInstall(t *testing.T) {
        cmd := exec.Command("helm", "install", "my-release", "local-repo/mychart")
        output, err := cmd.CombinedOutput()
    
        assert.NoError(t, err)
        assert.Contains(t, string(output), "Release \"my-release\" has been installed")
    }
    ```
    

----------

### **4. End-to-End Testing**

#### 4.1 **Set Up a Local Kubernetes Cluster**

-   Start a Minikube cluster:
    
    ```
    minikube start --driver=docker
    ```
    
-   Verify the Kubernetes cluster:
    
    ```
    kubectl get nodes
    ```
    

#### 4.2 **Simulate Helm Chart Installations**

-   Install a Helm Chart:
    
    ```
    helm install my-release local-repo/mychart
    ```
    
-   Verify Helm installation:
    
    ```
    helm list
    kubectl get deployments
    ```
    
-   Upgrade the Helm Chart:
    
    ```
    helm upgrade my-release local-repo/mychart
    ```
   -   Check upgrade status:
    
    ```
    helm status my-release
    kubectl get pods
    ```
    # Helm Chart Testing Guide


## **5. Mock External APIs for End-to-End Testing**

In end-to-end tests, external API calls might still need to be simulated. Mock servers help in replicating external services like token validation or metadata retrieval.

we can integrate the mock server (created during unit testing) with real Helm and Kubernetes interactions to test the following scenarios:

-   Downloading charts from a local registry
-   Performing token-based authentication
-   Interacting with Kubernetes services

### **Sample Go Test for End-to-End Helm Chart Installation**

Below is a Go test case that verifies Helm installation and Kubernetes deployment:

```go
`package e2e_test

import (
	"testing"
	"os/exec"
	"github.com/stretchr/testify/assert"
)

func TestHelmChartInstallE2E(t *testing.T) {
	// Ensure the Minikube cluster is up and Helm repo is added
	cmd := exec.Command("helm", "repo", "add", "local-repo", "file:///tmp/helm-repo")
	_, err := cmd.CombinedOutput()
	assert.NoError(t, err)

	// Install the chart
	cmd = exec.Command("helm", "install", "my-release", "local-repo/mychart")
	output, err := cmd.CombinedOutput()

	// Verify Helm installation
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Release \"my-release\" has been installed")

	// Check if Kubernetes deployment exists
	cmd = exec.Command("kubectl", "get", "deployments", "my-release")
	output, err = cmd.CombinedOutput()

	// Validate deployment in Kubernetes
	assert.NoError(t, err)
	assert.Contains(t, string(output), "my-release")
}`
```
