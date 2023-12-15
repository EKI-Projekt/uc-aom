// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package dbus

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/go-openapi/errors"
	log "github.com/sirupsen/logrus"
)

type DBusConnectionMock struct {
	reloadMessage string
	fakeWork      time.Duration
}

func NewDBusConnectionMock(reloadMessage string, fakeWork time.Duration) *DBusConnectionMock {
	return &DBusConnectionMock{reloadMessage: reloadMessage, fakeWork: fakeWork}
}

func (r DBusConnectionMock) Close() {}

func Initialize() Factory {
	return func(ctx context.Context) (DBusConnection, error) {
		return NewDBusConnectionMock("done", 0), nil
	}
}

func (r DBusConnectionMock) ReloadUnitContext(ctx context.Context, ch chan<- string) error {
	go func() {
		if r.fakeWork != 0 {
			time.Sleep(r.fakeWork)
		}

		nginxContainerName, err := getNginxContainerName()
		if err != nil {
			log.Errorf("ReloadUnitContext failed: %s", err.Error())
			ch <- err.Error()
			return
		}

		err = reloadNginxContainer(nginxContainerName)
		if err != nil {
			log.Errorf("ReloadUnitContext reloadNginxCommand failed: %s", err.Error())
			ch <- err.Error()
			return
		}
		ch <- r.reloadMessage
	}()

	return nil
}

func reloadNginxContainer(nginxContainerName string) error {
	reloadNginxCommand := exec.Command("docker", "container", "exec", nginxContainerName, "nginx", "-s", "reload")
	_, err := reloadNginxCommand.CombinedOutput()
	return err
}

func getNginxContainerName() (string, error) {
	containerNames, err := getAllRunningDockerContainerNames()
	if err != nil {
		return "", err
	}

	nginxContainerName := ""
	expectedNginxContainerSuffix := "uc-aom-nginx"
	for _, containerName := range containerNames {
		if strings.Contains(containerName, expectedNginxContainerSuffix) {
			nginxContainerName = containerName
			break
		}
	}

	if nginxContainerName != "" {
		return nginxContainerName, nil
	}

	return "", errors.NotFound(fmt.Sprintf("Docker container with suffix: %s", expectedNginxContainerSuffix))
}

func getAllRunningDockerContainerNames() ([]string, error) {
	getAllRunningDockerContainerNamesCommand := exec.Command("docker", "ps", "--format={{.Names}}")
	commandOut, err := getAllRunningDockerContainerNamesCommand.CombinedOutput()
	if err != nil {
		return nil, err
	}

	containerNames := strings.Split(string(commandOut), "\n")
	return containerNames, nil
}
