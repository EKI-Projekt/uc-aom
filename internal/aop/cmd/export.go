// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/registry"
	pkgRegistry "u-control/uc-aom/internal/pkg/registry"

	"github.com/spf13/cobra"
)

const exportExampleFmtStr = `%s export \
    --target-credentials <target credentials> \
	--version <app version> \
    --output <output directory>

The --target-credentials file has the same content as used by the push command, i.e:
{
	"username": "<username>",
	"password": "<password>",
	"repositoryname": "<repository name>"
}`

const exportFilenameSeparator = "_"

type exportOptions struct {
	credentialsFilepath string
	outputDirpath       string
	version             string
}

func NewExportCmd() *cobra.Command {
	exportOptions := exportOptions{}

	var exportCmd = &cobra.Command{
		Use:     "export",
		Short:   "Export app as an SWU file.",
		Example: fmt.Sprintf(exportExampleFmtStr, filepath.Base(os.Args[0])),
		RunE: func(cmd *cobra.Command, args []string) error {
			setLoggingVerbosity()
			return executeExportCommand(&exportOptions)
		},
	}

	exportCmd.Flags().StringVarP(&exportOptions.credentialsFilepath, "target-credentials", "t", "", "filepath to the registry credentials file. Must contain username, password and repositoryName")
	exportCmd.Flags().StringVarP(&exportOptions.outputDirpath, "output", "o", "", "output-directory where the pulled app will be stored as an swu-file. Default is current directory")
	exportCmd.Flags().StringVar(&exportOptions.version, "version", "", "version of the app to be pulled from the registry")

	exportCmd.MarkFlagRequired("target-credentials")
	exportCmd.MarkFlagRequired("version")

	return exportCmd
}

func executeExportCommand(exportOptions *exportOptions) error {
	ctx := context.Background()
	addOnTarget, err := getAddOnTarget(ctx, exportOptions.credentialsFilepath, exportOptions.version)
	if err != nil {
		return err
	}

	filename := getExportFilename(addOnTarget)

	exporter := packager.NewPackageExporter(fileio.CreateCpioArchive, fileio.Tarball)
	return exporter.Export(ctx, addOnTarget, exportOptions.outputDirpath, filename)
}

func getExportFilename(addOnTarget registry.AddOnRepositoryTarget) string {
	normalizedName := pkgRegistry.NormalizeCodeName(addOnTarget.AddOnRepository())
	return normalizedName + exportFilenameSeparator + addOnTarget.AddOnVersion() + ".swu"
}
