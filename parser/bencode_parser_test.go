package parser

import (
	"reflect"
	"testing"
)

func TestParseBenCode(t *testing.T) {
	tests := []struct {
		bencode      string
		expectedDict BencodeDict
	}{
		{
			bencode:      "de",
			expectedDict: make(BencodeDict),
		},
	}

	for _, test := range tests {
		res, err := ParseBencode(test.bencode)
		if err != nil {
			t.Errorf("failed to parse %s got this error instead %s\n", test.bencode, err)
		}

		if !reflect.DeepEqual(res, test.expectedDict) {
			t.Errorf("failed to parse %s expected %+v got %+v\n", test.bencode, test.expectedDict, res)
		}
	}
}

func TestConsumeInt(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		{
			input:       "i0e",
			expected:    0,
			expectError: false,
		},
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
		// should return error if it's not an empty int
		{
			input:       "ie",
			expectError: true,
		},
	}

	for _, test := range tests {
		i := 0
		res, err := consumeInt(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got ( %s ) as input", test.input)
		}

		if !test.expectError && err != nil {
			t.Errorf("was not expecting an error got ( %s ) instead", err)
		}

		if res != test.expected {
			t.Errorf("inputted ( %s ), expected ( %d ) got ( %d )", test.input, test.expected, res)
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
	}

	for _, test := range tests {
		i := 0
		res, err := consumeString(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got ( %s ) as input", test.input)
		}

		if !test.expectError && err != nil {
			t.Errorf("was not expecting an error got ( %s ) instead", err)
		}

		if res != test.expected {
			t.Errorf("inputted ( %s ), expected ( %s ) got ( %s )", test.input, test.expected, res)
		}
	}
}
