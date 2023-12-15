// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"fmt"
	"regexp"
	"sort"
	"u-control/uc-aom/internal/pkg/config"

	"github.com/hashicorp/go-version"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const addOnPackageVersionRegExpRaw = `-[0-9]+(-(alpha|beta|rc)\.[0-9]+)?`

var addOnPackageVersionRegExp *regexp.Regexp

func init() {
	addOnPackageVersionRegExp = regexp.MustCompile(addOnPackageVersionRegExpRaw)
}

// Interface type to sort a add-on version string array
type ByAddOnVersion []string

func (v ByAddOnVersion) Len() int      { return len(v) }
func (v ByAddOnVersion) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v ByAddOnVersion) Less(i, j int) bool {

	iAddOnPartnerVersion, err := v.createAddOnPartnerVersion(v[i])
	if err != nil {
		log.Tracef("Sort failed to interpret add-on partner version %v", iAddOnPartnerVersion)
		return false
	}
	jAddOnPartnerVersion, err := v.createAddOnPartnerVersion(v[j])
	if err != nil {
		log.Tracef("Sort failed to interpret add-on partner version %v", jAddOnPartnerVersion)
		return false
	}

	if iAddOnPartnerVersion.Equal(jAddOnPartnerVersion) {
		iAddOnPackageVersion, err := v.createAddOnPackageVersion(v[i])
		if err != nil {
			log.Tracef("Sort failed to interpret add-on package version %v", iAddOnPartnerVersion)
			return false
		}
		jAddOnPackageVersion, err := v.createAddOnPackageVersion(v[j])
		if err != nil {
			log.Tracef("Sort failed to interpret add-on package version %v", jAddOnPartnerVersion)
			return false
		}
		return iAddOnPackageVersion.LessThan(jAddOnPackageVersion)
	}

	return iAddOnPartnerVersion.LessThan(jAddOnPartnerVersion)
}

func (v ByAddOnVersion) createAddOnPartnerVersion(manifestVersion string) (*version.Version, error) {

	hasPackageVersion := addOnPackageVersionRegExp.Match([]byte(manifestVersion))

	if !hasPackageVersion {
		return &version.Version{}, fmt.Errorf("Partner version %s does not include a package version", manifestVersion)
	}

	versionParts := addOnPackageVersionRegExp.Split(manifestVersion, -1)
	addOnPartnerVersion := versionParts[0]
	return version.NewVersion(addOnPartnerVersion)
}

func (v ByAddOnVersion) createAddOnPackageVersion(manifestVersion string) (*version.Version, error) {
	addOnVersion := addOnPackageVersionRegExp.FindString(manifestVersion)
	if len(addOnVersion) > 1 {
		addOnVersion = addOnVersion[1:]
		return version.NewVersion(addOnVersion)
	}

	return &version.Version{}, fmt.Errorf("Can't find package version in %s", manifestVersion)
}

// check if the first version is greater or equal than the second version
func GreaterThanOrEqual(first string, second string) bool {

	if first == second {
		return true
	}

	return GreaterThan(first, second)
}

// check if the first version is greater than the second version
func GreaterThan(first string, second string) bool {
	versions := []string{first, second}
	sort.Sort(ByAddOnVersion(versions))
	return versions[0] != first
}

// Create annotations for the uc manifest layer descriptor
func CreateUcManifestAnnotationsV1_0(tag string, manifestVersion string) []string {

	return []string{
		ocispec.AnnotationTitle, config.UcImageLayerAnnotationTitle,
		ocispec.AnnotationVersion, tag,
		config.UcImageLayerAnnotationSchemaVersion, manifestVersion,
	}
}
