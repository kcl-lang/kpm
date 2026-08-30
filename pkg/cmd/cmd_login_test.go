package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
)

func TestLoginRejectsPasswordWithoutUsername(t *testing.T) {
	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	err = app.Run([]string{"kpm", "login", "-p", "aaaa", "ghcr.io"})
	assert.EqualError(t, err, "username must be specified when password is provided\n")
}

func TestLoginRejectsPasswordStdinWithoutUsername(t *testing.T) {
	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	err = app.Run([]string{"kpm", "login", "--password-stdin", "ghcr.io"})
	assert.EqualError(t, err, "username must be specified when password-stdin is used\n")
}

func TestLoginRejectsPasswordAndPasswordStdinTogether(t *testing.T) {
	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	err = app.Run([]string{"kpm", "login", "--password-stdin", "-u", "test", "-p", "aaaa", "ghcr.io"})
	assert.EqualError(t, err, "password and password-stdin cannot be used together\n")
}

func TestLoginRejectsUnknownProvider(t *testing.T) {
	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	err = app.Run([]string{"kpm", "login", "--provider=aws", "ghcr.io"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --provider=\"aws\"")
}

func TestLoginRejectsPasswordStdinWithGCPProvider(t *testing.T) {
	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	err = app.Run([]string{"kpm", "login", "--provider=gcp", "--password-stdin", "gcr.io/my-project"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--password-stdin has no effect with --provider=gcp")
}

func TestLoginGCPProvider_FakeMetadataServer(t *testing.T) {
	// Fake the GCE metadata server. The login command will mint a
	// token via the metadata client and then call LoginOci, which
	// will try to talk to the registry (also fake here).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instance/id"):
			_, _ = w.Write([]byte("1234567890123456789"))
		case strings.Contains(r.URL.Path, "/service-accounts/default/token"):
			_, _ = w.Write([]byte(`{"access_token":"ya29.fake","expires_in":3599,"token_type":"Bearer"}`))
		default:
			// Anything else (e.g. the OCI registry calls from
			// ORAS) — let the test server continue responding.
			w.WriteHeader(http.StatusOK)
		}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Point the metadata client at the test server. The metadata
	// package prepends "http://" itself, so the env var must be
	// just host:port.
	prev := os.Getenv("GCE_METADATA_HOST")
	parsed, err := url.Parse(srv.URL)
	require.NoError(t, err)
	require.NoError(t, os.Setenv("GCE_METADATA_HOST", parsed.Host))
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("GCE_METADATA_HOST")
		} else {
			_ = os.Setenv("GCE_METADATA_HOST", prev)
		}
	}()

	kpmcli, err := client.NewKpmClient()
	assert.NoError(t, err)

	app := &cli.App{
		Commands: []*cli.Command{
			NewLoginCmd(kpmcli),
		},
	}

	// We don't care that the OCI side fails (the fake server isn't
	// a real registry); we only care that the failure comes from
	// the OCI layer, not from the provider. So we just check that
	// the error message doesn't mention "not running on GCE/GKE"
	// (which would mean the provider never got to LoginOci).
	err = app.Run([]string{"kpm", "login", "--provider=gcp", srv.URL})
	if err != nil {
		assert.NotContains(t, err.Error(), "not running on GCE/GKE",
			"provider should have minted a token; OCI failure is expected but provider should not have failed")
	}
}
