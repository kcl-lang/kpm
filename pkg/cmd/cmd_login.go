// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/auth"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/reporter"
	"kcl-lang.io/kpm/pkg/utils"
)

// NewLoginCmd new a Command for `kpm login`.
func NewLoginCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "login",
		Usage:  "login to a registry",
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
			&cli.BoolFlag{
				Name:  "password-stdin",
				Usage: "take the registry password from stdin",
			},
			&cli.StringFlag{
				Name:  "provider",
				Usage: fmt.Sprintf("credential provider: %v (default \"basic\")", auth.KnownProviders()),
				Value: "basic",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return reporter.NewErrorEvent(
					reporter.InvalidCmd,
					fmt.Errorf("registry must be specified"),
				)
			}

			registry := c.Args().First()

			providerName := c.String("provider")
			if _, err := auth.ByName(providerName); err != nil {
				return reporter.NewErrorEvent(
					reporter.InvalidCmd,
					fmt.Errorf("invalid --provider=%q: %w", providerName, err),
				)
			}

			username, password, err := resolveCredentials(c, providerName)
			if err != nil {
				return err
			}

			// For non-basic providers the BasicProvider constructor
			// needs the resolved username/password. For basic the
			// caller already passed them in. For gcp the username/
			// password args are ignored — we build a GCPProvider
			// and call Credential() to mint a token.
			if providerName == "gcp" {
				gp := &auth.GCPProvider{}
				cred, err := gp.Credential(context.Background(), registry)
				if err != nil {
					return reporter.NewErrorEvent(
						reporter.FailedLogin,
						err,
						fmt.Sprintf("failed to obtain GCP credential for '%s'", registry),
					)
				}
				username = cred.Username
				password = cred.Password
			}

			err = kpmcli.LoginOci(registry, username, password)
			if err != nil {
				return err
			}

			// For non-basic providers, also record the host → provider
			// mapping in the sidecar so subsequent pulls know to mint
			// fresh credentials (the ORAS credential store only sees the
			// first token, which expires after ~1h for GCP).
			if providerName != "basic" && providerName != "" {
				store, storeErr := openProviderStoreFor(kpmcli)
				if storeErr != nil {
					return reporter.NewErrorEvent(
						reporter.FailedLogin,
						storeErr,
						fmt.Sprintf("failed to record provider mapping for '%s'", registry),
					)
				}
				if setErr := store.Set(registry, providerName); setErr != nil {
					return reporter.NewErrorEvent(
						reporter.FailedLogin,
						setErr,
						fmt.Sprintf("failed to record provider mapping for '%s'", registry),
					)
				}
			}

			reporter.ReportMsgTo("Login Succeeded", kpmcli.GetLogWriter())
			return nil
		},
	}
}

// openProviderStoreFor returns a ProviderStore rooted at the kpm
// settings directory. Used by login (writer) and logout (reader).
// We derive the path from settings so it always lives next to
// config.json regardless of how the kpm client was constructed.
func openProviderStoreFor(kpmcli *client.KpmClient) (*auth.ProviderStore, error) {
	// Settings.CredentialsFile is "<home>/.kpm/config/config.json".
	// The provider sidecar lives next to it as "providers.json".
	credPath := kpmcli.GetSettings().CredentialsFile
	if credPath == "" {
		return nil, fmt.Errorf("kpm settings have no CredentialsFile; cannot locate provider sidecar")
	}
	sidecarPath := filepath.Join(filepath.Dir(credPath), "providers.json")
	return auth.OpenProviderStore(sidecarPath), nil
}

// resolveCredentials resolves the (username, password) pair from CLI
// flags and stdin. For --provider=basic this is the existing flow. For
// --provider=gcp the username/password flags are ignored — we still
// error if the user passed incompatible flags like --password-stdin so
// we fail fast on misuse rather than silently ignoring them.
func resolveCredentials(c *cli.Context, providerName string) (string, string, error) {
	username := c.String("username")
	password := c.String("password")
	passwordStdin := c.Bool("password-stdin")

	if providerName == "gcp" {
		if passwordStdin {
			return "", "", reporter.NewErrorEvent(
				reporter.InvalidCmd,
				fmt.Errorf("--password-stdin has no effect with --provider=gcp; credentials are minted from the GCE/GKE metadata server"),
			)
		}
		if password != "" || username != "" {
			reporter.ReportMsgTo(
				"warning: --username and --password are ignored when --provider=gcp",
				c.App.Writer,
			)
		}
		return "", "", nil
	}

	if password != "" && passwordStdin {
		return "", "", reporter.NewErrorEvent(
			reporter.InvalidCmd,
			fmt.Errorf("password and password-stdin cannot be used together"),
		)
	}
	if password != "" && username == "" {
		return "", "", reporter.NewErrorEvent(
			reporter.InvalidCmd,
			fmt.Errorf("username must be specified when password is provided"),
		)
	}
	if passwordStdin && username == "" {
		return "", "", reporter.NewErrorEvent(
			reporter.InvalidCmd,
			fmt.Errorf("username must be specified when password-stdin is used"),
		)
	}

	return utils.GetUsernamePassword(username, password, passwordStdin)
}
