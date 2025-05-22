package client

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/mock"
	"os"
)

func TestLogin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping TestModCheckerCheck_WithTrustedSum test on Windows")
	}

	// Start the local Docker registry required for testing
	err := mock.StartDockerRegistry()
	assert.Equal(t, err, nil)

	defer func() {
		os.Unsetenv("OCI_REG_PLAIN_HTTP")
		err = mock.CleanTestEnv()
		if err != nil {
			t.Errorf("Error stopping docker registry: %v", err)
		}
	}()

	os.Setenv("OCI_REG_PLAIN_HTTP", "ON")
	kpmcli, err := NewKpmClient()
	assert.Equal(t, err, nil)
	err = kpmcli.LoginOci("172.88.0.8:5002", "test", "1234")
	assert.Equal(t, err, nil)
}
