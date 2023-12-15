package portainer

import (
	"regexp"
	"strings"
)

var rgx = regexp.MustCompile("[^a-z0-9]+")

// Strips all non-alphanumeric characters from name, coverts to lower case, finally returns the result.
func NormalizeName(name string) string {
	return rgx.ReplaceAllString(strings.ToLower(name), "")
}
