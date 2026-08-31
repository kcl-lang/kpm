// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewLogoutCmd new a Command for `kpm logout`.
func NewLogoutCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "logout",
		Usage:  "logout from a registry",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return reporter.NewErrorEvent(
					reporter.InvalidCmd,
					fmt.Errorf("registry must be specified"),
				)
			}
			hostname := c.Args().First()
			err := kpmcli.LogoutOci(hostname)
			if err != nil {
				return err
			}
			// Also drop any provider mapping for this host so future
			// pulls don't keep minting credentials via the provider
			// after the user logged out. This is a no-op when the
			// host wasn't registered with a non-basic provider.
			store, storeErr := openProviderStoreFor(kpmcli)
			if storeErr == nil {
				if delErr := store.Delete(hostname); delErr != nil {
					// Surface but don't fail the whole logout — the
					// ORAS credentials are already gone, which is
					// what the user asked for.
					reporter.ReportMsgTo(
						fmt.Sprintf("warning: failed to clear provider mapping for '%s': %v", hostname, delErr),
						kpmcli.GetLogWriter(),
					)
				}
			}
			reporter.ReportMsgTo("Logout Succeeded", kpmcli.GetLogWriter())
			return nil
		},
	}
}
