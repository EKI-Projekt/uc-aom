<!--
Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>

SPDX-License-Identifier: MIT
-->

# Developer guide to build and publish the uc-aom package using docker

See the [user guide](./USER_GUIDE.md) for information on how to use the `uc-aom-packager`.

## Build-time arguments used to version the uc-aom packager

We use docker to build and test, then finally publish the `uc-aom packager`.
If any unit tests fail then the build will fail, the `uc-aom packager` will not be published.

The following arguments can be provided when executing the docker build, all of which have defaults.
All of the arguments can be overridden at build-time using `--build-arg`, and will be used verbatim.
They all effect the version information of the `uc-aom packager` and any docker images it produces.
See the table below for details.
| Argument Name  | Default Value                                                             |
|----------------|---------------------------------------------------------------------------|
| BUILD_TIME     | Result of the `date` command in RFC 3339 format, e.g 2022-03-31T07:28:31Z |
| COPYRIGHT_YEAR | Result of the `date` command, resulting in a string similar to 2022       |
| VERSION        | v0.0.0                                                                    |

When executing `uc-aom packager` with the `--version` option, and assuming the default values from the table above, the following output will be displayed:

```text
> uc-aom-packager --version
UC-AOM PACKAGER v0.0.0
Built 2022-03-31T07:28:31Z
Copyright (C) 2022 Weidmüller Interface GmbH & Co. KG
```

Similarly the produced docker images have [layer annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md) to
identify the version of the `uc-aom packager` used.

### Example demonstrating the build-time override of the VERSION argument

It is anticipated that the VERSION argument will be overridden during continuous integration/delivery.
As an example, we could include the commit time and short GIT SHA1 hash:

```sh
docker buildx build --no-cache --load --tag uc-aop-latest --build-arg VERSION="v0.0.0-${git rev-parse --short HEAD}" -f deployments/Dockerfile ../../
```

## Building and publishing the uc-aom packager via docker

To make the `uc-aom-packager` tool available as a docker image, build and publish it to the `wmucdev` Azure container registry,
by performing the following:

```sh
docker login --username "<username>" --password "<password>" "wmucdev.azurecr.io"
docker buildx build \
    --no-cache \
    --platform linux/amd64 \
    --push -t wmucdev.azurecr.io/u-control/uc-aom-packager:v0.0.0 \
    --build-arg VERSION=v0.0.0 \
    -f assets/Dockerfile .
```
