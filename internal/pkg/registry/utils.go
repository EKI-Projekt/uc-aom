package registry

import "strings"

// codenameSeparator separates the codename and appname if the app is published within a certain codename.
// example: posuma/test-uc-addon-status-running
const codenameSeparator = "/"

// NormalizeCodeName removes the codename from the repository and returns a normalized repository without the codename.
func NormalizeCodeName(repository string) string {
	split := strings.Split(repository, codenameSeparator)
	hasCodeName := len(split) > 1
	if hasCodeName {
		return split[len(split)-1]
	}
	return repository
}
