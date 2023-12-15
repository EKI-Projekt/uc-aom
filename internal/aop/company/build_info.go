// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package company

import (
	"fmt"
	"time"
)

// Replaced at build-time.
var (
	buildtime     string
	version       string
	copyrightyear string
)

// company author name
const AuthorName = "Weidmüller Interface GmbH & Co. KG"

const versionDefault = "v0.0.0"
const shortProductInfoFmtStr = "%s. UC-AOM PACKAGER %s (%s)"
const versionInfoFmtStr = `UC-AOM PACKAGER %s
Built %s
Copyright (C) %s %s.
`

// Returns the build time of the packager as defined by RFC 3339.
// See: https://www.ietf.org/rfc/rfc3339.txt
func BuildTime() string {
	if len(buildtime) == 0 {
		buildtime = time.Now().Format(time.RFC3339)
	}
	return buildtime
}

// Returns the version of the packager.
func Version() string {
	if len(version) == 0 {
		version = versionDefault
	}
	return version
}

// Returns the copyright year of the packager.
func CopyrightYear() string {
	if len(copyrightyear) == 0 {
		copyrightyear = fmt.Sprintf("%d", time.Now().Year())
	}
	return copyrightyear
}

// Returns the version text of the packager
func VersionWithCopyrightNotice() string {
	return fmt.Sprintf(versionInfoFmtStr, Version(), BuildTime(), CopyrightYear(), AuthorName)
}

// Returns a short oneline information that can be used to identify this product.
func ShortAuthorInfo() string {
	return fmt.Sprintf(shortProductInfoFmtStr, AuthorName, Version(), BuildTime())
}
