package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"u-control/uc-aom/internal/aom/utils"
	model "u-control/uc-aom/internal/pkg/manifest"
	modelV0_1 "u-control/uc-aom/internal/pkg/manifest/v0_1"
)

type versionResolver interface {
	getVersion() (string, error)
	updateVersion(version string) error
}

const releaseFilename = "release.json"

type releaseFile struct {
	Version string `json:"version"`
}

type aomVersionResolver struct {
	localStateDir   string
	localFSRegistry localFSRegistry
}

func (r *aomVersionResolver) getVersion() (string, error) {
	releaseFilePath := path.Join(r.localStateDir, releaseFilename)
	fileContent, err := os.ReadFile(releaseFilePath)

	if errors.Is(err, os.ErrNotExist) {
		installedRepositories, err := r.localFSRegistry.Repositories()
		if err != nil {
			return "", err
		}

		if len(installedRepositories) == 0 {
			err = r.updateVersion(currentVersion)
			return currentVersion, err
		}

		resolvedVersion, err := r.resolveVersionBasedOn(installedRepositories)
		if err != nil {
			return "", err
		}
		err = r.updateVersion(resolvedVersion)
		return resolvedVersion, err
	}

	if err != nil {
		return "", err
	}

	releaseFile := &releaseFile{}
	err = json.Unmarshal(fileContent, releaseFile)
	if err != nil {
		return "", err
	}

	return releaseFile.Version, nil
}

func (r *aomVersionResolver) resolveVersionBasedOn(repositories []string) (string, error) {
	for _, repositoryName := range repositories {
		repository, err := r.localFSRegistry.Repository(repositoryName)
		if err != nil {
			return "", fmt.Errorf("Unexpected error while get repository: %v.", err)
		}

		manifestVersion, _, err := fetchManifestFrom(repository)

		if err != nil {
			return "", fmt.Errorf("Unexpected error while fetch manifest: %v.", err)
		}

		var resolvedVersion string
		switch manifestVersion {
		case modelV0_1.ValidManifestVersion:
			resolvedVersion = "0.3.2"
		case model.ValidManifestVersion:
			resolvedVersion = "0.4.0"
		default:
			return "", fmt.Errorf("Unknown version manifest version %s.", manifestVersion)
		}

		return resolvedVersion, nil

	}
	return "", errors.New("Could not resolve version from installed add-ons.")

}

func (r *aomVersionResolver) updateVersion(version string) error {
	newReleaseFile := &releaseFile{
		Version: version,
	}
	fileContent, err := json.Marshal(newReleaseFile)
	if err != nil {
		return err
	}
	return utils.WriteFileToDestination(releaseFilename, fileContent, r.localStateDir)
}
