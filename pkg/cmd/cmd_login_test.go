package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
