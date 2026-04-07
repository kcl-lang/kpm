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
