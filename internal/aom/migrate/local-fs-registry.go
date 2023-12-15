package migrate

import (
	"io"
	"os"
	"path"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/config"
)

type localFSRepository interface {
	// Fetch fetches the content of the repository.
	Fetch() (io.Reader, error)

	// Push pushes the content to the repository.
	Push(content io.Reader) error
}

type localFSRegistry interface {
	// Return all names of add-on repositories
	Repositories() ([]string, error)

	// Repository returns a repository reference by the given name.
	Repository(name string) (localFSRepository, error)
}

type localFSRegistryAdapter struct {
	root    string
	localfs *manifest.LocalFSRepository
}

func (a *localFSRegistryAdapter) Repositories() ([]string, error) {
	return a.localfs.GetManifestsDirectories(a.root)
}

func (a *localFSRegistryAdapter) Repository(name string) (localFSRepository, error) {
	return &localFSRepositoryAdapter{root: a.root, repository: name}, nil
}

type localFSRepositoryAdapter struct {
	root       string
	repository string
}

func (r *localFSRepositoryAdapter) Fetch() (io.Reader, error) {
	path := r.createInstallFilePathWith(config.UcImageManifestFilename)
	return os.Open(path)
}

func (r *localFSRepositoryAdapter) Push(content io.Reader) error {
	tempFilename := "." + config.UcImageManifestFilename
	tempFilePath := r.createInstallFilePathWith(tempFilename)
	file, err := os.Create(tempFilePath)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, content)
	if err != nil {
		return err
	}
	return os.Rename(tempFilePath, r.createInstallFilePathWith(config.UcImageManifestFilename))
}

func (r *localFSRepositoryAdapter) createInstallFilePathWith(filename string) string {
	return path.Join(r.root, r.repository, filename)
}
