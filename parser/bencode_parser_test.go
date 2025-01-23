package parser

import (
	"reflect"
	"testing"
)

// func TestParseBenCode(t *testing.T) {
// 	tests := []struct {
// 		bencode      string
// 		expectedDict BencodeDict
// 	}{
// 		{
// 			bencode:      "de",
// 			expectedDict: make(BencodeDict),
// 		},
// 	}

// 	for _, test := range tests {
// 		res, err := ParseBencode(test.bencode)
// 		if err != nil {
// 			t.Errorf("failed to parse %s got this error instead %s\n", test.bencode, err)
// 		}

// 		if !reflect.DeepEqual(res, test.expectedDict) {
// 			t.Errorf("failed to parse %s expected %+v got %+v\n", test.bencode, test.expectedDict, res)
// 		}
// 	}
// }

func TestConsumeList(t *testing.T) {
	tests := []struct {
		input       string
		expected    []any
		expectError bool
	}{
		// should work as expected for empty list
		{
			input:       "le",
			expected:    []any{},
			expectError: false,
		},
		// should work as expected for single item list
		{
			input:       "li5ee",
			expected:    []any{5},
			expectError: false,
		},
		// should work as expected for list of diff items
		{
			input:       "li5ei32e1:he",
			expected:    []any{5, 32, "h"},
			expectError: false,
		},
		// should work as expected for nested lists
		{
			input:       "li3el1:hee",
			expected:    []any{3, []any{"h"}},
			expectError: false,
		},
		{
			input:       "lli5ee1:se",
			expected:    []any{[]any{5}, "s"},
			expectError: false,
		},
		// should throw if an item in the list is wrong
		{
			input:       "l1:hhhe",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "l1:hhhe",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "l",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "e",
			expected:    nil,
			expectError: true,
		},
	}

	for _, test := range tests {
		i := 0
		res, err := consumeList(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got '%s' as input %d", test.input, i)
		}

		if !test.expectError && err != nil {
			t.Errorf("was not expecting an error got '%s' instead", err)
		}

		if !reflect.DeepEqual(res, test.expected) {
			t.Errorf("inputted '%s', expected ( %+v ) got ( %+v )", test.input, test.expected, res)
		}
	}
}

func TestConsumeInt(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		// should work as expected for single digits number
		{
			input:       "i0e",
			expected:    0,
			expectError: false,
		},
		// should work as expected for multiple digits number
		{
			input:       "i555555e",
			expected:    555555,
			expectError: false,
		},
		{
			input:       "i-12e",
			expected:    -12,
			expectError: false,
		},
		// should return error if it's not int
		{
			input:       "istringe",
			expectError: true,
		},
		// should return error if it's an empty int
		{
			input:       "ie",
			expectError: true,
		},
		// should return error if it's invalid int
		{
			input:       "5",
			expectError: true,
		},
		{
			input:       "i",
			expectError: true,
		},
		{
			input:       "e",
			expectError: true,
		},
	}

	for _, test := range tests {
		i := 0
		res, err := consumeInt(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got '%s' as input", test.input)
		}

		if !test.expectError && err != nil {
			t.Errorf("was not expecting an error got '%s' instead", err)
		}

		if res != test.expected {
			t.Errorf("inputted '%s', expected '%d' got '%d'", test.input, test.expected, res)
		}
	}
}

func TestConsumeString(t *testing.T) {
	tests := []struct {
		input, expected string
		expectError     bool
	}{
		// should work as expected
		{
			input:       "1:h",
			expected:    "h",
			expectError: false,
		},
		// should parse only specified str len
		{
			input:       "1:hh",
			expected:    "h",
			expectError: false,
		},
		// should return an error if len is neg
		{
			input:       "-1:hh",
			expectError: true,
		},
		// should return an error if len is zero
		{
			input:       "0:tt",
			expectError: true,
		},
		// should return an error if str is invalid
		{
			input:       "0",
			expectError: true,
		},
		{
			input:       ":ttt",
			expectError: true,
		},
		{
			input:       "1f",
			expectError: true,
		},
	}

	for _, test := range tests {
		i := 0
		res, err := consumeString(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got '%s' as input", test.input)
		}

		if !test.expectError && err != nil {
			t.Errorf("was not expecting an error got '%s' instead", err)
		}

		if res != test.expected {
			t.Errorf("inputted '%s', expected '%s' got '%s'", test.input, test.expected, res)
		}
	}
}
