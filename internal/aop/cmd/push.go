// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"u-control/uc-aom/internal/aop/credentials"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/manifest"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/aop/utils"
	"u-control/uc-aom/internal/pkg/config"
	model "u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
)

const pushExampleFmtStr = `%s push \
    --manifest <manifest directory> \
    --source-credentials <source credentials> \
    --target-credentials <target credentials> \
    -v

The --source-credentials file is required to access the registry that hosts the docker images which are referenced in the app's manifest.json file.
It has the following format:
{
    "username": "<username>",
    "password": "<password>",
    "serveraddress": "<registry URL>"
}

The --target-credentials file will be provided and has the following format:
{

    "username": "<username>",
    "password": "<password>",
    "repositoryname": "<repository name>",
	"serveraddress": "<registry URL>"
}`

// Use the logo regex from the manifest schema to validate the logo app file
var logoRegexp = regexp.MustCompile("^.+\\.(jpg|png|jpeg|svg)$")

func filterAppFilesFunc(fileName string) bool {
	return fileName == config.UcImageManifestFilename || logoRegexp.MatchString(fileName)
}

type pushOptions struct {
	sourceCredentialsFilepath string
	targetCredentialsFilepath string
	contentDirpath            string
}

func NewPushCommand() *cobra.Command {
	pushOptions := pushOptions{}
	var pushCmd = &cobra.Command{
		Use:          "push",
		Short:        "Create and publish an app package.",
		Example:      fmt.Sprintf(pushExampleFmtStr, filepath.Base(os.Args[0])),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setLoggingVerbosity()
			return executePushCommand(&pushOptions)
		},
	}
	pushCmd.Flags().StringVarP(&pushOptions.sourceCredentialsFilepath, "source-credentials", "s", "", "path to a file providing credentials, for the registry hosting docker images, referenced in the app's manifest.json file")
	pushCmd.Flags().StringVarP(&pushOptions.targetCredentialsFilepath, "target-credentials", "t", "", "path to a file providing credentials for the app's host repository. The target registry is always the Weidmüller development registry")
	pushCmd.Flags().StringVarP(&pushOptions.contentDirpath, "manifest", "m", "", "a directory path which contains logo and manifest.json files")

	pushCmd.MarkFlagRequired("source-credentials")
	pushCmd.MarkFlagRequired("target-credentials")
	pushCmd.MarkFlagRequired("manifest")

	return pushCmd
}

func executePushCommand(pushOptions *pushOptions) error {
	sourceCredentials, err := credentials.ParseAndValidate(os.ReadFile, pushOptions.sourceCredentialsFilepath, credentials.ServerAddressSet())
	if err != nil {
		log.Errorf("Invalid credentials file '%s': %v", pushOptions.sourceCredentialsFilepath, err)
		return err
	}

	targetCredentials, err := credentials.ParseAndValidate(os.ReadFile, pushOptions.targetCredentialsFilepath, credentials.RepositoryNameSet())
	if err != nil {
		log.Errorf("Invalid credentials file '%s': %v", pushOptions.targetCredentialsFilepath, err)
		return err
	}

	credentials.SetRegistryServerAddress(targetCredentials)
	ctx := context.Background()

	source, err := registry.InitializeRegistry(ctx, sourceCredentials)
	if err != nil {
		log.Errorf("Could not initialize docker image source registry: %v", err)
		return err
	}

	target, err := registry.InitializeRegistry(ctx, targetCredentials)
	if err != nil {
		log.Errorf("Could not initialize app host registry: %v", err)
		return err
	}

	log.Info("Initializing repository.")
	repo, err := target.Repository(ctx, targetCredentials.RepositoryName)
	if err != nil {
		log.Errorf("Could not initialize app host repository: %v", err)
		return err
	}

	log.Info("Initializing working directory.")
	workDir, err := os.MkdirTemp("", "aop-workdir-*")
	if err != nil {
		log.Errorf("Could not initialize work directory: %v", err)
		return err
	}
	defer os.RemoveAll(workDir)

	err = utils.CopyFiles(pushOptions.contentDirpath, workDir, filterAppFilesFunc)

	if err != nil {
		log.Errorf("Could not copy to working directory: %v", err)
		return err
	}

	log.Info("Creating app package.")
	err = manifest.MigrateManifestFile(workDir)
	if err != nil {
		log.Errorf("Could not migrate manifest: %v", err)
		return err
	}

	OsManifestFileReader := model.NewOsManifestFileReader()
	addOnManifest, err := manifest.ParseAndValidate(OsManifestFileReader, os.Stat, workDir)
	if err != nil {
		log.Errorf("Could not parse manifest: %v", err)
		return err
	}

	packager := packager.NewPackageCreator(fileio.GzipTarball, registry.ExportUsingContainerd)
	artifact, err := packager.CreateOciArtifact(ctx, addOnManifest, source)
	if err != nil {
		log.Errorf("Could not create app package: %v", err)
		return err
	}

	log.Info("Pushing app package to repository.")
	_, err = registry.Copy(ctx, artifact, repo, oras.DefaultCopyOptions)
	if err != nil {
		log.Errorf("Could not push app package to repository: %v", err)
		return err
	}

	log.Infoln("Succeeded!")
	return nil
}
