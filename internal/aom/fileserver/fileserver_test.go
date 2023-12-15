// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileserver_test

import (
	"errors"
	"io"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/fileserver"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

type registryMock struct {
	mock.Mock
}

func (m registryMock) Tags(repository string) ([]string, error) {
	args := m.Called(repository)
	return args.Get(0).([]string), args.Error(1)
}

func (m registryMock) Repositories() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m registryMock) Pull(repository string, tag string, processor registry.ImageManifestLayerProcessor) (uint64, error) {
	args := m.Called(repository, tag, processor)
	return args.Get(0).(uint64), args.Error(1)
}

func (m registryMock) Delete(repository string, tag string) error {
	args := m.Called(repository, tag)
	return args.Error(0)
}

type swUpdateWatcherMock struct {
	mock.Mock
}

func (s *swUpdateWatcherMock) Connect() error {
	args := s.Called()
	return args.Error(0)
}

func (s *swUpdateWatcherMock) ListenOnStatus() (<-chan fileserver.RecoveryStatus, <-chan error) {
	msgChan := make(chan fileserver.RecoveryStatus)
	errChan := make(chan error)
	go func() {
		time.Sleep(1 * time.Millisecond)
		errChan <- nil
		msgChan <- fileserver.SUCCESS
	}()
	return msgChan, errChan
}

type swUpdateWatcherWithErrosMock struct {
	mock.Mock
	connectCallsCount int
}

func (s *swUpdateWatcherWithErrosMock) Connect() error {
	log.Info("return error", s.connectCallsCount)
	if s.connectCallsCount == 1 {
		return errors.New("Connection Error")
	}
	s.connectCallsCount = 1
	return nil
}

func (s *swUpdateWatcherWithErrosMock) ListenOnStatus() (<-chan fileserver.RecoveryStatus, <-chan error) {
	msgChan := make(chan fileserver.RecoveryStatus)
	errChan := make(chan error)
	go func() {
		time.Sleep(1 * time.Millisecond)
		errChan <- errors.New("io.EOF")
		msgChan <- fileserver.PROGRESS
	}()
	return msgChan, errChan
}

type addOnCreatorMock struct {
	mock.Mock
}

func (a *addOnCreatorMock) CreateAddOnRoutine(repository string, version string) error {
	args := a.Called(repository, version)
	return args.Error(0)

}

func (a *addOnCreatorMock) UpdateAddOnRoutine(repository string, version string, settings ...*manifest.Setting) error {
	var args mock.Arguments
	if len(settings) > 0 {
		args = a.Called(repository, version, settings)
	} else {
		args = a.Called(repository, version)
	}
	return args.Error(0)
}

func createUut(t *testing.T, watcher fileserver.SWUpdateWatcher, registry registry.AddOnRegistry, localCatalogue catalogue.LocalAddOnCatalogue) (*fileserver.FileServer, *service.ServiceMultiComponentMock) {
	mockObj := &service.ServiceMultiComponentMock{}
	serviceStub := mockObj.NewServiceUsingServiceMultiComponentMock()
	transactionResolver := service.NewTransactionScheduler()
	fs := fileserver.NewFileServerWithTransactionSchedular(transactionResolver, watcher, serviceStub, registry, localCatalogue)

	return fs, mockObj
}

func TestFileServer_InstallOnSwuSuccess(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	watcher.On("Connect").Return(nil)

	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{}, nil)

	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, _ := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)

	// act
	wg, _ := uut.StartSWUpdateWatcher()
	time.Sleep(10 * time.Millisecond)
	uut.StopSWUpdateWatcher()

	// assert
	watcher.AssertExpectations(t)
	dropInRegistryMock.AssertExpectations(t)
	wg.Wait()
}

func TestFileServer_ConnectError(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherWithErrosMock{
		connectCallsCount: 0,
	}

	dropInRegistryMock := &registryMock{}
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, _ := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)

	// act
	wg, _ := uut.StartSWUpdateWatcher()
	wg.Wait()

	// assert
	if watcher.connectCallsCount != 1 {
		t.Errorf("Expectd to have call connect twice")
	}
}

func TestFileServer_InstallAllDropInAddOns(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"xyz"}, nil)
	dropInRegistryMock.On("Delete", "abc", "xyz").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	addon := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "abc",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "xyz",
				Platform:        []string{"ucm"},
			},
			Version: "xyz",
		},
		DockerImageData: []io.Reader{},
	}
	ts.On("PullAddOn", "abc", "xyz").Return(addon, nil)
	ts.On("Validate", mock.Anything).Return(nil)
	ts.MockStackService.On("CreateStackWithDockerCompose", "abc", mock.AnythingOfType("string")).Return(nil)
	ts.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	ts.On("AvailableSpaceInBytes").Return(uint64(101), nil)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsContinueOnTagsError(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{}, errors.New(""))
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, _ := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsContinueOnTagsEmtry(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{}, nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, _ := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsWith2Repos(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc", "def"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"xyz"}, nil)
	dropInRegistryMock.On("Tags", "def").Return([]string{"xyz"}, nil)
	dropInRegistryMock.On("Delete", "abc", "xyz").Return(nil)
	dropInRegistryMock.On("Delete", "def", "xyz").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	ts.On("GetAddOn", "def").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	addonAbc := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "abc",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "xyz",
				Platform:        []string{"ucm"},
			},
			Version: "xyz",
		},
		DockerImageData: []io.Reader{},
	}
	addonDef := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "def",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "xyz",
				Platform:        []string{"ucm"},
			},
			Version: "xyz",
		},
		DockerImageData: []io.Reader{},
	}
	ts.On("PullAddOn", "abc", "xyz").Return(addonAbc, nil)
	ts.On("PullAddOn", "def", "xyz").Return(addonDef, nil)
	ts.On("Validate", mock.Anything).Return(nil).Twice()
	ts.MockStackService.On("CreateStackWithDockerCompose", "abc", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	ts.MockStackService.On("CreateStackWithDockerCompose", "def", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	ts.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil).Twice()
	ts.On("AvailableSpaceInBytes").Return(uint64(101), nil).Twice()

	// act
	uut.InstallAllDropInAddOns()

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsNewTransaction(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"xyz"}, nil)
	dropInRegistryMock.On("Delete", "abc", "xyz").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	addon := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "abc",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "xyz",
				Platform:        []string{"ucm"},
			},
			Version: "xyz",
		},
		DockerImageData: []io.Reader{},
	}
	ts.On("PullAddOn", "abc", "xyz").Return(addon, nil)
	ts.On("Validate", mock.Anything).Return(nil)
	ts.MockStackService.On("CreateStackWithDockerCompose", "abc", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	ts.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	ts.On("AvailableSpaceInBytes").Return(uint64(101), nil)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsUpdateGreaterVersionTransaction(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"0.2.0-1"}, nil)
	dropInRegistryMock.On("Delete", "abc", "0.2.0-1").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil).Once()
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil).Twice()
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	ts.On("DeleteAddOn", "abc").Return(nil)
	addon := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "abc",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "0.2.0-1",
				Platform:        []string{"ucm"},
			},
			Version: "0.2.0-1",
		},
		DockerImageData: []io.Reader{},
	}
	ts.MockStackService.On("DeleteAddOnStack", "abc").Return(nil)
	ts.MockStackService.On("RemoveUnusedVolumes", "abc", mock.Anything).Return(nil)
	ts.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	ts.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	ts.On("PullAddOn", "abc", "0.2.0-1").Return(addon, nil)
	ts.On("Validate", mock.Anything).Return(nil)
	ts.On("FetchManifest", addon.AddOn.Name, addon.AddOn.Version).Return(&addon.AddOn.Manifest, nil)
	ts.MockStackService.On("CreateStackWithDockerCompose", "abc", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	ts.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	ts.On("AvailableSpaceInBytes").Return(uint64(1), nil)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsUpdateGreaterVersionWithSettingsTransaction(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"0.2.0-1"}, nil)
	dropInRegistryMock.On("Delete", "abc", "0.2.0-1").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil)
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil).Twice()
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	ts.On("DeleteAddOn", "abc").Return(nil)
	env := make(map[string]string)
	env["param1"] = "aaa"
	ts.On("GetAddOnEnvironment", "abc").Return(env, nil)
	settingsBefore := make(map[string][]*manifest.Setting)
	envSettingsBefore := make([]*manifest.Setting, 2)
	envSettingsBefore[0] = manifest.NewSettings("param1", "param1", false).WithTextBoxValue("bbb")
	envSettingsBefore[1] = manifest.NewSettings("param3", "param3", false).WithTextBoxValue("xyz")
	settingsBefore["environmentVariables"] = envSettingsBefore
	addon := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "abc",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         "0.2.0-1",
				Platform:        []string{"ucm"},
				Settings:        settingsBefore,
			},
			Version: "0.2.0-1",
		},
		DockerImageData: []io.Reader{},
	}
	ts.MockStackService.On("DeleteAddOnStack", "abc").Return(nil)
	ts.MockStackService.On("RemoveUnusedVolumes", "abc", mock.Anything).Return(nil)
	ts.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	ts.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	ts.On("PullAddOn", "abc", "0.2.0-1").Return(addon, nil)
	ts.On("Validate", mock.Anything).Return(nil)
	ts.On("FetchManifest", addon.AddOn.Name, "0.2.0-1").Return(&addon.AddOn.Manifest, nil)
	envSettingsAfter := make([]*manifest.Setting, 2)
	envSettingsAfter[0] = manifest.NewSettings("param1", "param1", false).WithTextBoxValue("aaa")
	envSettingsAfter[1] = manifest.NewSettings("param3", "param3", false).WithTextBoxValue("xyz")
	ts.MockStackService.On("CreateStackWithDockerCompose", "abc", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	ts.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	ts.On("AvailableSpaceInBytes").Return(uint64(2), nil)

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsUpdateSmallerVersionTransaction(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"0.1.0-1"}, nil)
	dropInRegistryMock.On("Delete", "abc", "0.1.0-1").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.2.0-1"}, nil)
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil).Once()

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}

func TestFileServer_InstallAllDropInAddOnsUpdateEqualVersionTransaction(t *testing.T) {
	// arrange
	watcher := &swUpdateWatcherMock{}
	dropInRegistryMock := &registryMock{}
	dropInRegistryMock.On("Repositories").Return([]string{"abc"}, nil)
	dropInRegistryMock.On("Tags", "abc").Return([]string{"0.2.0-1"}, nil)
	dropInRegistryMock.On("Delete", "abc", "0.2.0-1").Return(nil)
	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.2.0-1"}, nil)
	uut, ts := createUut(t, watcher, dropInRegistryMock, localCatalogueMock)
	ts.On("GetAddOn", "abc").Return(catalogue.CatalogueAddOn{Name: "abc", Version: "0.1.0-1"}, nil).Once()

	// act
	err := uut.InstallAllDropInAddOns()
	if err != nil {
		t.Errorf("Expect not error but got: %v", err)
	}

	// assert
	dropInRegistryMock.AssertExpectations(t)
	localCatalogueMock.AssertExpectations(t)
	ts.AssertExpectations(t)
}
