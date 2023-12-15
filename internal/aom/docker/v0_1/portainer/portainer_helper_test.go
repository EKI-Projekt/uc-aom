package portainer_test

import (
	"testing"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer"
)

func TestNormalizeName(t *testing.T) {
	var names [3]string

	names[0] = "Anyviz Cloud Adapter (VPN)"
	names[1] = "anyviz-cloud-adapter-VPN"
	names[2] = "Anyviz 'Cloud' Adapter (VPN)"

	expectedName := "anyvizcloudadaptervpn"

	for _, name := range names {
		normalizedName := portainer.NormalizeName(name)
		if normalizedName != expectedName {
			t.Errorf("Expected name to be normalized to %s but got %s", expectedName, name)
		}
	}
}
