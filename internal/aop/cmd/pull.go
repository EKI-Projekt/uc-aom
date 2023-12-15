// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"u-control/uc-aom/internal/aop/credentials"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/registry"
)

const pullExampleFmtStr = `%s pull \
    --target-credentials <target credentials> \
    --output <output directory> \
    --version <app version> \
    -v

The --target-credentials file has the same content as used by the push command, i.e:
{
    "username": "<username>",
    "password": "<password>",
    "repositoryname": "<repository name>",
	"serveraddress": "<registry URL>"
}`

type pullOptions struct {
	credentialsFilepath string
	outputDirpath       string
	version             string
	extract             bool
}

// pullCmd represents the pull command
func NewPullCmd() *cobra.Command {
	pullOptions := pullOptions{}
	var pullCmd = &cobra.Command{
		Use:     "pull",
		Short:   "Pull an app package from the registry.",
		Example: fmt.Sprintf(pullExampleFmtStr, filepath.Base(os.Args[0])),
		RunE: func(cmd *cobra.Command, args []string) error {
			setLoggingVerbosity()
			return executePullCommand(&pullOptions)
		},
	}

	pullCmd.Flags().StringVarP(&pullOptions.credentialsFilepath, "target-credentials", "t", "", "filepath to the target registry credentials file. The target registry is always the Weidmüller development registry")
	pullCmd.Flags().StringVarP(&pullOptions.outputDirpath, "output", "o", "", "output directory where the pulled app will be stored")
	pullCmd.Flags().StringVar(&pullOptions.version, "version", "", "version of the app to be pulled from the registry")
	pullCmd.Flags().BoolVarP(&pullOptions.extract, "extract", "x", true, "extract the app archive")

	pullCmd.MarkFlagRequired("target-credentials")
	pullCmd.MarkFlagRequired("output")
	pullCmd.MarkFlagRequired("version")

	return pullCmd
}

func executePullCommand(pullOptions *pullOptions) error {
	_, err := os.Stat(pullOptions.outputDirpath)
	if err != nil {
		return fmt.Errorf("Could not access the output path %s: %v", pullOptions.outputDirpath, err)
	}

	ctx := context.Background()
	addOnTarget, err := getAddOnTarget(ctx, pullOptions.credentialsFilepath, pullOptions.version)

	packagerReader := packager.NewPackageReader(fileio.UnGzipTarball)
	_, err = packagerReader.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: pullOptions.outputDirpath, Extract: pullOptions.extract})
	return err
}

func getAddOnTarget(ctx context.Context, credentialsFilepath string, version string) (registry.AddOnRepositoryTarget, error) {
	addOnHostCredentials, err := credentials.ParseAndValidate(os.ReadFile, credentialsFilepath, credentials.RepositoryNameSet())
	if err != nil {
		return nil, fmt.Errorf("Invalid credentials file '%s': %v", credentialsFilepath, err)
	}

	credentials.SetRegistryServerAddress(addOnHostCredentials)
	log.Info("Initializing registries.")
	addOnHostRegistry, err := registry.InitializeRegistry(ctx, addOnHostCredentials)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize app host registry: %v", err)
	}

	addOnRepository, err := addOnHostRegistry.Repository(ctx, addOnHostCredentials.RepositoryName)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize app host repository: %v", err)
	}
	addOnTarget := registry.NewOciRepositoryTargetDecorator(addOnRepository, version, addOnHostCredentials.RepositoryName)
	return addOnTarget, nil
}
