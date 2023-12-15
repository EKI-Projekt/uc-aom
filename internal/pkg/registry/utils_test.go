package registry_test

import (
	"fmt"
	"testing"
	"u-control/uc-aom/internal/pkg/registry"
)

func TestNormalizeCodeName(t *testing.T) {
	type args struct {
		repositotyName string
		normalizedName string
	}

	testCases := []args{
		{
			repositotyName: "posuma/test-uc-addon",
			normalizedName: "test-uc-addon",
		},
		{
			repositotyName: "test-uc-addon",
			normalizedName: "test-uc-addon",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Should normalize %s to be %s", tc.repositotyName, tc.normalizedName), func(t *testing.T) {
			// Act
			gotName := registry.NormalizeCodeName(tc.repositotyName)

			// Assert
			if gotName != tc.normalizedName {
				t.Errorf("Expected normalized name to be %s but got %s", tc.normalizedName, gotName)
			}
		})
	}
}
