// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/env"
	"u-control/uc-aom/internal/aom/fileserver"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/iam"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/migrate"
	"u-control/uc-aom/internal/aom/network"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/service"
	addon_status "u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/aom/system"
	sharedConfig "u-control/uc-aom/internal/pkg/config"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type UcAom struct {
	grpcListener net.Listener
}

func NewUcAom(grpcListener net.Listener) *UcAom {
	return &UcAom{grpcListener}
}

func (u *UcAom) Setup() error {
	err := os.MkdirAll(sharedConfig.CACHE_DROP_IN_PATH, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(sharedConfig.PERSISTENCE_DROP_IN_PATH, os.ModePerm)
	if err != nil {
		return err
	}

	_, err = os.Stat(catalogue.ASSETS_INSTALL_PATH)
	if err != nil {
		return err
	}

	_, err = os.Stat(catalogue.ASSETS_TMP_PATH)
	if err != nil {
		return err
	}

	return nil
}

func (u *UcAom) Run() error {
	grpc_server := grpc.NewServer()
	localfs := manifest.NewRepository(os.ReadFile, filepath.WalkDir)

	registryCredentials, credError := registry.GetRegistryCredentials()
	if credError != nil {
		log.Fatalf("Error %v", credError)
		return credError
	}
	orasRegistry, err := registry.InitializeRegistry(registryCredentials)
	if err != nil {
		log.Fatalf("Unable to initialize ORAS registry: %v", err)
	}

	orasRemoteRegistry := registry.NewORASAddOnRegistry(orasRegistry, localfs, runtime.GOARCH, runtime.GOOS)
	addOnRegistry := registry.NewCodeNameAdapterRegistry(orasRemoteRegistry)
	orasRemote := catalogue.NewORASRemoteAddOnCatalogue(catalogue.ASSETS_TMP_PATH, addOnRegistry, localfs)
	localCatalogue := catalogue.NewLocalAddOnCatalogue(catalogue.ASSETS_INSTALL_PATH, addOnRegistry, localfs)

	writeToFile := func(name string, writeContent func(io.Writer) error) error {
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		return writeContent(f)
	}

	deleteFile := func(name string) error {
		if _, err := os.Stat(name); err == nil {
			log.Debugf("Path '%s' exists, removing...", name)
			err = os.Remove(name)
			if err != nil {
				log.Trace("Remove failed!")
				return err
			}
		}
		return err
	}

	createSymbolicLink := func(target string, linkname string) error {
		log.Debugf("ln -s %s %s", target, linkname)
		return os.Symlink(target, linkname)
	}

	reverseProxy := routes.NewReverseProxy(dbus.Initialize(), routes.SITES_AVAILABLE_PATH, routes.SITES_ENABLED_PATH, routes.ROUTES_MAP_AVAILABLE_PATH, routes.ROUTES_MAP_ENABLED_PATH, writeToFile, deleteFile, createSymbolicLink, deleteFile)

	iamServiceUcAomClient := iam.NewIamServiceClient(iam.IAM_AUTH_URL, iam.IAM_AUTH_SERVICE_UCAOM, iam.IAM_AUTH_ENDPOINT)

	iamServiceUcAuthClient := iam.NewIamServiceClient(iam.IAM_AUTH_URL, iam.IAM_AUTH_SERVICE_UCAUTH, iam.IAM_AUTH_ENDPOINT)

	dockerCli, err := docker.NewDockerCli()
	if err != nil {
		return err
	}

	composeService := compose.NewComposeService(dockerCli)
	stackService := docker.NewStackService(dockerCli.Client(), composeService)

	err = service.StopInstalledAddOns(localCatalogue, stackService)
	if err != nil {
		return err
	}

	addOnStatusResolver := addon_status.AdaptAddOnStatusResolverToDocker(stackService)

	addOnEnvResolver := env.NewAddOnEnvironmentResolver(stackService)

	iamPermissionWriter := iam.NewIamPermissionWriter(iam.IAM_PERMISSION_PATH, writeToFile, deleteFile)

	transactionScheduler := service.NewTransactionScheduler()

	manifestValidator, err := model.NewValidator()
	if err != nil {
		return err
	}
	uOSSystem := system.NewuOSSystem(catalogue.ASSETS_INSTALL_PATH)

	adminUser, err := uOSSystem.LookupAdminUser()
	if err != nil {
		return err
	}
	err = docker.CreateAndServelocalPublicAccessVolumesDriver(adminUser)
	if err != nil {
		return err
	}

	err = docker.CreateAndServeLocalPublicVolumesDriver(adminUser)
	if err != nil {
		return err
	}
	service := service.NewService(stackService, reverseProxy, iamPermissionWriter, localCatalogue, manifestValidator, addOnEnvResolver, uOSSystem)
	err = migrateInstalledAddOns(transactionScheduler, service, stackService, localfs, addOnEnvResolver, reverseProxy)
	if err != nil {
		return err
	}
	err = startAllInstalledAddOns(localCatalogue, stackService, dockerCli.Client())
	if err != nil {
		return err
	}

	installAllDropInAddOnsInPersistenceFolder(transactionScheduler, localfs, stackService, reverseProxy, iamPermissionWriter, manifestValidator, addOnEnvResolver, uOSSystem)

	fileServer := createFileServer(transactionScheduler, localfs, stackService, reverseProxy, iamPermissionWriter, manifestValidator, addOnEnvResolver, uOSSystem)

	err = fileServer.InstallAllDropInAddOns()
	if err != nil {
		return err
	}
	_, err = fileServer.StartSWUpdateWatcher()
	if err != nil {
		return err
	}

	grpc_api.RegisterAddOnServiceServer(grpc_server,
		server.NewServer(service, config.URL_ASSETS_LOCAL_ROOT, config.URL_ASSETS_REMOTE_ROOT, localCatalogue, orasRemote, iamServiceUcAomClient, iamServiceUcAuthClient, addOnStatusResolver, addOnEnvResolver, transactionScheduler))

	log.Infof("Server is listening on %s ...", u.grpcListener.Addr().String())
	return grpc_server.Serve(u.grpcListener)
}

func installAllDropInAddOnsInPersistenceFolder(transactionScheduler *service.TransactionScheduler, localfs *manifest.LocalFSRepository, stackService *docker.StackService, reverseProxy *routes.ReverseProxy, iamPermissionWriter *iam.IamPermissionWriter, validator model.Validator, addOnEnvironmentResolver *env.AddOnEnvironmentResolver, system system.System) {
	swUpdateWatcher := fileserver.NewSWUpdateWatcher("/tmp/swupdateprog")
	dropInAddOnRegistry := registry.NewDropInAddOnRegistry(sharedConfig.PERSISTENCE_DROP_IN_PATH, runtime.GOARCH, runtime.GOOS, &manifest.ManifestTarGzipDecompressor{})
	localCatalogue := catalogue.NewLocalAddOnCatalogue(catalogue.ASSETS_INSTALL_PATH, dropInAddOnRegistry, localfs)
	serviceForDropIn := service.NewService(stackService, reverseProxy, iamPermissionWriter, localCatalogue, validator, addOnEnvironmentResolver, system)
	fileServer := fileserver.NewFileServerWithTransactionSchedular(transactionScheduler, swUpdateWatcher, serviceForDropIn, dropInAddOnRegistry, localCatalogue)
	fileServer.InstallAllDropInAddOns()
}

func createFileServer(transactionScheduler *service.TransactionScheduler, localfs *manifest.LocalFSRepository, stackService *docker.StackService, reverseProxy *routes.ReverseProxy, iamPermissionWriter *iam.IamPermissionWriter, validator model.Validator, addOnEnvironmentResolver *env.AddOnEnvironmentResolver, system system.System) *fileserver.FileServer {
	swUpdateWatcher := fileserver.NewSWUpdateWatcher("/tmp/swupdateprog")
	dropInAddOnRegistry := registry.NewDropInAddOnRegistry(sharedConfig.CACHE_DROP_IN_PATH, runtime.GOARCH, runtime.GOOS, &manifest.ManifestTarGzipDecompressor{})
	localCatalogue := catalogue.NewLocalAddOnCatalogue(catalogue.ASSETS_INSTALL_PATH, dropInAddOnRegistry, localfs)
	serviceForDropIn := service.NewService(stackService, reverseProxy, iamPermissionWriter, localCatalogue, validator, addOnEnvironmentResolver, system)
	fileServer := fileserver.NewFileServerWithTransactionSchedular(transactionScheduler, swUpdateWatcher, serviceForDropIn, dropInAddOnRegistry, localCatalogue)
	return fileServer
}

func startAllInstalledAddOns(localCatalogue catalogue.LocalAddOnCatalogue, stackService docker.StackServiceAPI, dockerCli client.APIClient) error {
	internalBridgeConnector := network.NewInternalBridgeNetworkConnector(dockerCli)
	starter := service.NewAddOnStarter(localCatalogue, stackService, internalBridgeConnector)
	return starter.StartInstalledAddOns()

}

func migrateInstalledAddOns(transactionScheduler *service.TransactionScheduler,
	service *service.Service, stackService docker.StackServiceAPI, localfs *manifest.LocalFSRepository, envResolver env.EnvResolver, reverseProxy *routes.ReverseProxy) error {
	migrator := migrate.NewInstallAddOnMigrator(catalogue.ASSETS_INSTALL_PATH, localfs, transactionScheduler, service, stackService, envResolver, reverseProxy)
	return migrator.Migrate()
}
