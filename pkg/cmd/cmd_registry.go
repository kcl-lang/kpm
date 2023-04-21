// Copyright 2023 The KCL Authors. All rights reserved.

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moby/term"
	"github.com/urfave/cli/v2"
	"kusionstack.io/kpm/pkg/oci"
	"kusionstack.io/kpm/pkg/reporter"
	"kusionstack.io/kpm/pkg/settings"
)

// NewRegistryCmd new a Command for `kpm registry`.
func NewRegCmd(settings *settings.Settings) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "registry",
		Usage:  "run registry command.",
		Subcommands: []*cli.Command{
			{
				Name:  "login",
				Usage: "login to a registry",
				Flags: []cli.Flag{
					// The registry username.
					&cli.StringFlag{
						Name:    "username",
						Aliases: []string{"u"},
						Usage:   "registry username",
					},
					// The registry registry password or identity token.
					&cli.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "registry password or identity token",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 {
						reporter.Report("kpm: registry must be specified.")
						reporter.ExitWithReport("kpm: run 'kpm registry help' for more information.")
					}
					registry := c.Args().First()

					username, password, err := getUsernamePassword(c.String("username"), c.String("password"), c.Bool("password-stdin"))
					if err != nil {
						return err
					}

					err = oci.Login(registry, username, password, settings)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "logout from a registry",
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 {
						reporter.Report("kpm: registry must be specified.")
						reporter.ExitWithReport("kpm: run 'kpm registry help' for more information.")
					}
					registry := c.Args().First()

					err := oci.Logout(registry, settings)
					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}
}

// Copied/Adapted from https://github.com/helm/helm
func getUsernamePassword(usernameOpt string, passwordOpt string, passwordFromStdinOpt bool) (string, string, error) {
	var err error
	username := usernameOpt
	password := passwordOpt

	if password == "" {
		if username == "" {
			username, err = readLine("Username: ", false)
			if err != nil {
				return "", "", err
			}
			username = strings.TrimSpace(username)
		}
		if username == "" {
			password, err = readLine("Token: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", errors.New("token required")
			}
		} else {
			password, err = readLine("Password: ", true)
			if err != nil {
				return "", "", err
			} else if password == "" {
				return "", "", errors.New("password required")
			}
		}
	}

	return username, password, nil
}

// Copied/adapted from https://github.com/helm/helm
func readLine(prompt string, silent bool) (string, error) {
	fmt.Print(prompt)
	if silent {
		fd := os.Stdin.Fd()
		state, err := term.SaveState(fd)
		if err != nil {
			return "", err
		}
		err = term.DisableEcho(fd, state)
		if err != nil {
			return "", err
		}

		defer func() {
			restoreErr := term.RestoreTerminal(fd, state)
			if err == nil {
				err = restoreErr
			}
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if silent {
		fmt.Println()
	}

	return string(line), nil
}
