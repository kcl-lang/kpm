# KPM Test Framework — Design

**Issue**: #593 (LFX), #621 (offline test infra)

## Problem

KPM today has two parallel test setups:

1. **Real network tests** — `pkg/client/client_test.go` hits `ghcr.io`, `docker.io`, etc. directly. They
   are slow, flaky, blocked by firewalls, and cannot run in CI sandboxes without registry credentials.
2. **Docker-script tests** — `pkg/mock/oci_env_mock.go` boots a local Docker registry via shell scripts
   (`scripts/reg.sh`, `pkg/mock/test_script/push_pkg.sh`). These work, but they need Docker installed,
   can't run in parallel without port collisions, and the "test pass" depends on the scripts not silently
   failing.

What KPM needs is a **pure-Go, in-process, offline-friendly** mock OCI registry that exercises the
real downloader code path (`pkg/oci`, `pkg/downloader`) without external dependencies.

## Goals

1. **No network**: tests run with `KPM_REG` pointing at `127.0.0.1:<random-port>`.
2. **No Docker / no scripts**: the mock registry is a Go `httptest.Server` wrapping an in-memory
   `oras.land/oras-go/v2/content/memory.Store`.
3. **Familiar ergonomics**: drop-in helpers so existing test files just call
   `testruntime.Start(t)` instead of `mock.StartDockerRegistry()`.
4. **Reproducible fixtures**: helper to seed a memory store from a `.tar` fixture so the same package
   blob is used by every test.

## Non-goals (this PR)

- Replacing every existing Docker-based test. (Migration is follow-up work; this PR lands the building
  block.)
- Replacing unit tests with mocks. (Unit tests stay as they are.)
- A new CLI subcommand. (Out of scope — the issue says "simple CLI" but the immediate value is in the
  Go API; CLI work is a separate follow-up.)

## Design

### Layout

```
pkg/testruntime/
  runtime.go       // Start(t) starts an in-memory OCI registry on a random port
  fixture.go       // LoadTarFixture / PushPackage helpers to seed the registry
  client.go        // Returns a *OciClient pointed at the test registry
  runtime_test.go  // Smoke test: round-trip a fixture through the registry
```

### `runtime.go`

```go
// Start boots an in-memory OCI registry on 127.0.0.1:<random-port> and
// returns a handle that:
//   - registers a t.Cleanup() to shut down the registry,
//   - exposes RegistryURL() so callers can set KPM_REG / OciOptions.Registry,
//   - exposes Store() for tests that want to push fixtures directly.
func Start(t *testing.T, opts ...Option) *Runtime
```

Internally it wraps `oras.land/oras-go/v2/registry/remote.NewRepository` against an
`httptest.Server` that serves the OCI distribution API in front of a
`oras.land/oras-go/v2/content/memory.Store`. The mock is *not* a full distribution-spec server — it
implements only the routes KPM hits (`/v2/<name>/manifests/<ref>`, `/v2/<name>/blobs/<digest>`).

### `fixture.go`

```go
// LoadTarFixture loads an OCI layout fixture (.tar produced by `kcl mod push`)
// into the test runtime's memory store under the given repo + tag.
func LoadTarFixture(t *testing.T, rt *Runtime, repo, tag, tarPath string)
```

This lets us reuse the `test_data/*.tar` artifacts that the existing e2e suite already commits,
without having to regenerate fixtures.

### Migration plan (follow-up)

1. Convert `pkg/client/test_load_pkg_from_oci` to use the mock instead of the network.
2. Convert `pkg/visitor/...` tests that currently shell out to Docker.
3. Keep `pkg/mock/oci_env_mock.go` as a wrapper for any test that still needs Docker (e.g. testing
   the Docker registry fallback path itself).

## Alternatives considered

- **`go-containerregistry` registry mock**: heavier API surface; covers more of the distribution spec
  than we need.
- **Plain `httptest` + hand-written handlers**: would work but reinvents what oras-go already
  provides via `content/memory.Store`.
- **`testcontainers-go`**: still requires Docker, just better ergonomics.

## Open questions

- Should we expose a `kpm test --offline` subcommand that runs `go test ./...` with `KPM_REG`
  pre-pointed at a long-lived shared mock? Nice-to-have, but a separate PR.
- How do we model auth? The current network tests ignore auth. Mock auth is not in scope here.