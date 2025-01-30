package encoder

import (
	"gotorrent/decoder"
	"testing"
)

func TestEncodeDict(t *testing.T) {
	// maps in go are not ordered, meaning iterating through the keys and encoding them
	// will not produce the same result upon different re-runs
	// thus the test here a quite simple
	// @TODO: maybe think of a way to test with an unordered dict?
	tests := []struct {
		input    decoder.BencodeDict
		expected string
	}{
		{
			input:    decoder.BencodeDict{},
			expected: "de",
		},
		{
			input:    decoder.BencodeDict{"h": "h"},
			expected: "d1:h1:he",
		},
		{
			input:    decoder.BencodeDict{"h": 1},
			expected: "d1:hi1ee",
		},
		{
			input:    decoder.BencodeDict{"h": []any{1}},
			expected: "d1:hli1eee",
		},
		{
			input:    decoder.BencodeDict{"h": decoder.BencodeDict{"h": "h"}},
			expected: "d1:hd1:h1:hee",
		},
	}

	for _, test := range tests {
		result := encodeDict(test.input)
		if result != test.expected {
			t.Errorf("input = %+v expected %s = , got = %s", test.input, test.expected, result)
		}
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		input    []any
		expected string
	}{
		{
			input:    []any{},
			expected: "le",
		},
		{
			input:    []any{"hi", 1},
			expected: "l2:hii1ee",
		},
		{
			input:    []any{"hi", 1, []any{1}},
			expected: "l2:hii1eli1eee",
		},
		{
			input:    []any{decoder.BencodeDict{"h": "h"}},
			expected: "ld1:h1:hee",
		},
	}

	for _, test := range tests {
		result := encodeList(test.input)
		if result != test.expected {
			t.Errorf("input = %+v expected %s = , got = %s", test.input, test.expected, result)
		}
	}
}

func TestEncodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "0:",
		},
		{
			input:    "0",
			expected: "1:0",
		},
	}

	for _, test := range tests {
		result := encodeString(test.input)
		if result != test.expected {
			t.Errorf("input = %s expected %s = , got = %s", test.input, test.expected, result)
		}
	}
}

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{
			input:    5,
			expected: "i5e",
		},
		{
			input:    0,
			expected: "i0e",
		},
		{
			input:    -1,
			expected: "i-1e",
		},
	}

	for _, test := range tests {
		result := encodeInt(test.input)
		if result != test.expected {
			t.Errorf("input = %d expected %s = , got = %s", test.input, test.expected, result)
		}
	}
}
