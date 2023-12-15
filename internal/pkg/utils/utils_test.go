package utils_test

import (
	"fmt"
	"os"
	"testing"
	"u-control/uc-aom/internal/pkg/utils"
)

func TestGetEnv(t *testing.T) {
	type args struct {
		key           string
		value         string
		fallBack      string
		expectedValue string
	}

	testCases := []args{
		{key: "TEST_VALUE", value: "abc", fallBack: "xyz", expectedValue: "abc"},
		{key: "TEST_VALUE", value: "", fallBack: "xyz", expectedValue: "xyz"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should have the value of %s", tc.key, tc.expectedValue), func(t *testing.T) {
			// Arrange
			if tc.value != "" {
				os.Setenv(tc.key, tc.value)
				t.Cleanup(func() {
					os.Unsetenv(tc.key)
				})
			}

			// Act
			res := utils.GetEnv(tc.key, tc.fallBack)

			// Assert
			if res != tc.expectedValue {
				t.Errorf("Expected result to be %s but got %s", tc.fallBack, res)
			}
		})
	}
}
