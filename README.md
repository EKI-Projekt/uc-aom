<!--
Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>

SPDX-License-Identifier: MIT
-->

# uc-aom

app manager for installing custom user apps on the u-control.
This repository hosts the backend service/daemon (`uc-aomd`), the command line tool to communicate with the daemon (`uc-aom`) and a command line tool to publish apps (`uc-aop`) to a container registry.

## Table of contents

[[_TOC_]]

## Maintainers

This project is currently maintained by:

- Michael Brockmeyer (w010261)
- Felix Aldenburg (w100130)

Please contact the persons above for merge requests or any questions regarding this repository.

## Prerequisites in production
- Install and start [SWUpdate](https://sbabic.github.io/swupdate/swupdate.html) to install apps via a swu file.
- Install and start Nginx as a reverse proxy to integrate app web UIs.
- Internet connection to have access to wmucrel.azurecr.io as [OCI](https://github.com/opencontainers/image-spec/blob/main/spec.md) registry

## Development Setup

The preferred method to build and run the service (`uc-aomd`), CLI (`uc-aom`) and publishing tool (`uc-aop`) for development is using Docker.
Only a Linux host is supported as a development platform, if on Windows you can install a linux distribution using WSL2.

The source code is organized according to the open source [GO standard project-layout](https://github.com/golang-standards/project-layout) recommendations.

### Docker based development environment

#### Quick start

Open [Visual Studio Code](https://code.visualstudio.com/) and trigger the _reopen in container_ command to start the environment in development mode.

#### Script entry points

The _reopen in container_ will start the environment in development mode which should be sufficient for test and development.

However, other operation modes are provided via the script `./script/docker-run.sh`.
As an example, to build then run the `uc-aomd` backend service, gRPC web proxy, SWUpdate instance, docker registry and an [NGINX](http://nginx.org/) server:

```shell
./scripts/docker-run.sh start
```

Once the development environment is up and running it is possible to attach to the `uc-aom` docker container.

The development environment has the following services which have been mapped to the host once started.

| Service         | Host URL       | Description                                                                                          |
| --------------- | -------------- | ---------------------------------------------------------------------------------------------------- |
| uc-aom          | localhost:3800 | Provides a gRPC endpoint which can be debugged with [evans](https://github.com/ktr0731/evans).       |
| Docker Registry | localhost:5000 | Access to the [Docker Registry HTTP API V2](https://docs.docker.com/registry/spec/api/).             |
| NGINX           | localhost:8080 | [NGINX reverse proxy and http server](http://nginx.org/).                                            |
| SWUpdate        | localhost:8090 | [SWUpdate](https://sbabic.github.io/swupdate/) instance to develop/test the offline install feature. |

#### uc-aomd | backend service

Open this folder in VS code, the **Run and Debug** settings have been configured to launch the `uc-aomd` backend service.
The testing extension is also pre-configured to enable you to run the tests.

Contained in the root folder is also a Makefile which offers several tasks, including cross-compiling the backend service for the u-control.
To view all tasks the Makefile has to offer, refer to its dedicated help via `make help`.

#### uc-aom | CLI tool

The CLI can be build via the [Makefile](./Makefile) which offers multiple variants to build and crosscompile the CLI tool e.g. `make build-cli` or `make build-device-cli`.

To use the CLI tool call `uc-aom --help` to show all available commands.

#### uc-aop | app publishing tool

The **Run and Debug** settings have been configured to launch the `uc-aop` tool which will publish a test app to the docker registry running as part of the development environment.
The testing extension is also pre-configured to enable you to run the tests.

The `tools/uc-aop` folder contains a dedicated [Makefile](./tools/uc-aop/Makefile) which offers several tasks, including building the `uc-aop` command line tool.
To view all tasks the Makefile has to offer, refer to its dedicated help via `make help`.

More information is contained in the [developer README.md](./tools/uc-aop/README.md) and [USER_GUIDE.md](./tools/uc-aop/USER_GUIDE.md) files, respectively.

## Publishing the test apps to wmucdev Azure container registry

This repository contains a set of dedicated test apps to facilitate development and test of the app manager.
The `./scripts/docker-run.sh` script supports publishing these apps to the `wmucdev` Azure container registry via the following command:

```shell
./scripts/docker-run.sh tc-apps
```

This will build and publish _all_ test apps contained in the `./testdata` folder.

## Repository for sharing the documentation with app partners

We are using a private [github repository](https://github.com/weidmueller/uc-addon/settings) to share our documentation with our partners.
Each partner get will get access to this repository.

The [manifest documentation](./api/uc-manifest.schema-doc.md) and [app packager user guide](./tools/uc-aop/USER_GUIDE.md) need to be provided for each new release.
We need to create a corresponding tag in the Github repository after we release new documentation.
The tag must be the same as that used in this repository and for the app packager's docker image.

## Reuse-Tool (check and add Copyright and License information)

To check current Copyright andd Licence informaiton you can use the "reuse-tool". For detailed information see [Usage - reuse](https://reuse.readthedocs.io/en/stable/index.html).

### Check files for Copyright and License headers

Open a shell into the running docker Container and run

```shell
reuse lint
```

this will list all files without valid Copyright and License-headers.

### Add Copyright ans License headers to file

Open a shell into the running docker Container and run

```shell
reuse addheader --copyright="Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>" --license=MIT <file-path>
```

## Migration test
To execute the migration test follow these steps:
1.  `./scripts/migration-test.sh up`
    - The command has to be executed outside of the development environment. This prepares the test environment with different versions of the uc-aom and uc-aop.
2.  `./scripts/migration-test.sh run`
    - This command starts the test itself. Executing the test may take some time.
3.  `./scripts/migration-test.sh down`
    - This command removes the migration test setup.

# Releases

This table shows which uc-aom version was released in which specific device releases.

uc-aom version | Device release |
| :--- | :----   |
| 0.3.2  | 1.16.0+ |
| 0.4.0  | 2.0.0  |
| 0.5.0-rc.4  | 2.0.1  |
| 0.5.0  | 1.17.1  |
| 0.5.2  | 1.17.2  |
