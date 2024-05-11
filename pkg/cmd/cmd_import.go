// Copyright 2023 The KCL Authors. All rights reserved.
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"kcl-lang.io/kcl-go/pkg/tools/gen"
	"kcl-lang.io/kpm/pkg/client"
	"kcl-lang.io/kpm/pkg/reporter"
)

// NewImportCmd new a Command for `kpm import`.
func NewImportCmd(kpmcli *client.KpmClient) *cli.Command {
	return &cli.Command{
		Hidden: false,
		Name:   "import",
		Usage:  "convert other formats to KCL file",
		Description: `import converts other formats to KCL file.

	Supported conversion modes:
	json:            convert JSON data to KCL data
	yaml:            convert YAML data to KCL data
	gostruct:        convert Go struct to KCL schema
	jsonschema:      convert JSON schema to KCL schema
	terraformschema: convert Terraform schema to KCL schema
	auto:            automatically detect the input format`,
		ArgsUsage: "<file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Aliases:     []string{"m"},
				Usage:       "mode of import",
				DefaultText: "auto",
				Value:       "auto",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output filename",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "force overwrite output file",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() != 1 {
				return fmt.Errorf("invalid arguments")
			}
			inputFile := c.Args().First()

			opt := &gen.GenKclOptions{}
			switch c.String("mode") {
			case "json":
				opt.Mode = gen.ModeJson
			case "yaml":
				opt.Mode = gen.ModeYaml
			case "gostruct":
				opt.Mode = gen.ModeGoStruct
			case "jsonschema":
				opt.Mode = gen.ModeJsonSchema
			case "terraformschema":
				opt.Mode = gen.ModeTerraformSchema
			case "auto":
				opt.Mode = gen.ModeAuto
			default:
				return fmt.Errorf("invalid mode: %s", c.String("mode"))
			}

			outputFile := c.String("output")
			if outputFile == "" {
				outputFile = "generated.k"
				reporter.ReportMsgTo("output file not specified, use default: generated.k", kpmcli.GetLogWriter())
			}

			if _, err := os.Stat(outputFile); err == nil && !c.Bool("force") {
				return fmt.Errorf("output file already exist, use --force to overwrite: %s", outputFile)
			}

			outputWriter, err := os.Create(outputFile)
			if err != nil {
				return reporter.NewErrorEvent(reporter.FailedCreateFile, err, fmt.Sprintf("failed to create output file: %s", outputFile))
			}

			return gen.GenKcl(outputWriter, inputFile, nil, opt)
		},
	}
}
