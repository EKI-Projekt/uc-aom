// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"u-control/uc-aom/internal/pkg/manifest"
)

func TestVersion(t *testing.T) {
	type testCaseData struct {
		input    []string
		expected []string
	}
	testCases := []testCaseData{
		{
			[]string{"0.9.9.3-alpha.1-1", "0.8.9.1-1", "0.9.9.3-1", "0.9.9.3-beta.1-1", "0.9.9.9-1", "1.0.0-rc.4-1"},
			[]string{"0.8.9.1-1", "0.9.9.3-alpha.1-1", "0.9.9.3-beta.1-1", "0.9.9.3-1", "0.9.9.9-1", "1.0.0-rc.4-1"},
		},
		{
			[]string{"0.9.9.3-1-alpha.1", "0.8.9.1-1", "0.9.9.3-1", "0.9.9.3-1-beta.1", "0.9.9.9-1", "1.0.0-1-rc.4"},
			[]string{"0.8.9.1-1", "0.9.9.3-1-alpha.1", "0.9.9.3-1-beta.1", "0.9.9.3-1", "0.9.9.9-1", "1.0.0-1-rc.4"},
		},
		{
			[]string{"1.0.0-1", "0.9.9-1", "1.0.0-rc.9-1", "0.1.1-1"},
			[]string{"0.1.1-1", "0.9.9-1", "1.0.0-rc.9-1", "1.0.0-1"},
		},
		{
			[]string{"1.0.0-1", "1.0.0-1"},
			[]string{"1.0.0-1", "1.0.0-1"},
		},
		{
			[]string{"1.0.0", "1.0.0-", "1.0.0-1", "-1"},
			[]string{"1.0.0", "1.0.0-", "1.0.0-1", "-1"},
		},
		{
			[]string{"0.4-rc.5-1", "0.2-beta.1-1", "0.1-1", "0.5-1", "0.2-1"},
			[]string{"0.1-1", "0.2-beta.1-1", "0.2-1", "0.4-rc.5-1", "0.5-1"},
		},
		{
			[]string{"HKO_2.3-45", "HKO_1.2-34", "HKO_3.4-56"},
			[]string{"HKO_2.3-45", "HKO_1.2-34", "HKO_3.4-56"},
		},
		{
			[]string{"1.9.9.3-alpha.1-1", "2.8.9.1-1", "1.9.9.3-1", "0.9.9.3-beta.1-1", "0.9.9.9-1", "3.0.0-rc.4-1"},
			[]string{"0.9.9.3-beta.1-1", "0.9.9.9-1", "1.9.9.3-alpha.1-1", "1.9.9.3-1", "2.8.9.1-1", "3.0.0-rc.4-1"},
		},
		{
			[]string{"1.9.9.3-1-alpha.1", "2.8.9.1-1", "1.9.9.3-1", "0.9.9.3-1-beta.1", "0.9.9.9-1", "3.0.0-1-rc.4"},
			[]string{"0.9.9.3-1-beta.1", "0.9.9.9-1", "1.9.9.3-1-alpha.1", "1.9.9.3-1", "2.8.9.1-1", "3.0.0-1-rc.4"},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("Sorting #[%s]", strings.Join(testCase.input, "|")), func(t *testing.T) {
			// Arrange
			actual := testCase.input

			// Act
			sort.Sort(manifest.ByAddOnVersion(actual))

			// Assert
			if !reflect.DeepEqual(actual, testCase.expected) {
				t.Errorf("Sort failed.\n\tExpected: %#v.\n\tActual: %#v", testCase.expected, actual)
			}
		})
	}
}

func TestGreaterThanOrEqual(t *testing.T) {
	type testCaseData struct {
		first    string
		second   string
		expected bool
	}
	testCases := []testCaseData{
		{
			"0.9.9.3-alpha.1-1",
			"0.8.9.1-1",
			true,
		},
		{
			"0.9.9.3-1",
			"0.9.9.3-beta.1-1",
			true,
		},
		{
			"0.9.9.9-1",
			"1.0.0-rc.4-1",
			false,
		},
		{
			"0.8.9.1-1",
			"0.9.9.3-alpha.1-1",
			false,
		},
		{
			"0.9.9.3-beta.1-1",
			"0.9.9.3-1",
			false,
		},
		{
			"0.9.9.9-1",
			"1.0.0-rc.4-1",
			false,
		},
		{
			"1.0.0-1",
			"1.0.0-rc.4-1",
			true,
		},
		{
			"1.0.0-1",
			"1.0.0-1",
			true,
		},
		{
			"1.0.0-1",
			"1.0.0-1-rc.1",
			true,
		},
		{
			"1.0.0-rc.2-1",
			"1.0.0-rc.1-1",
			true,
		},
		{
			"1.0.0-1-rc.1",
			"1.0.0-1",
			false,
		},
	}

	for _, testCase := range testCases {
		testCaseName := fmt.Sprintf("manifest.GreaterThanOrEqual() first %s second %s expected %t", testCase.first, testCase.second, testCase.expected)
		t.Run(testCaseName, func(t *testing.T) {
			// Arrange & Act
			result := manifest.GreaterThanOrEqual(testCase.first, testCase.second)

			// Assert
			if result != testCase.expected {
				t.Errorf("Failed manifest.GreaterThanOrEqual(): want %t, got %t", testCase.expected, result)
			}
		})
	}
}
