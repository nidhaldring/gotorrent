package decoder

import (
	"reflect"
	"testing"
)

func TestParseBenCode(t *testing.T) {
	// @TODO: write better tests
	expectedResult := &TorrentFile{
		Announce: "https://torrent.ubuntu.com/announce",
		AnnounceList: []any{
			[]any{"https://torrent.ubuntu.com/announce"},
			[]any{"https://ipv6.torrent.ubuntu.com/announce"},
		},
		CreatedBy:    "mktorrent 1.1",
		CreationDate: 1724947415,
		Encoding:     "UTF-8", // Assuming you want to keep this field
		Info: TorrentInfo{
			Length:      6203355136,
			Name:        "ubuntu-24.04.1-desktop-amd64.iso",
			PieceLength: 262144,
			// Pieces: "", // Uncomment and populate if needed
		},
	}

	// @TODO: this is here to silence "unused vars"
	_ = expectedResult

	testFilePath := "./files/test.torrent"
	parsed, err := DecodeTorrentFile(testFilePath)
	if err != nil {
		t.Fatalf("Expected ParseBencode to return result got error %s instead", err)
	}

	if parsed == nil {
		t.Fatal("Expected result to be different to nil")
	}

  // @TODO: uncomment this!
	// @TODO: this currently fails because "Pieces" field is too large
	// to put in the struct and i want to find a better way to put it there
	// if parsed != nil && !reflect.DeepEqual(expectedResult, parsed) {
	// t.Fatalf("Expected results to be equal inputted file '%s'", testFilePath)
	// }
}

func TestConsumeDict(t *testing.T) {
	tests := []struct {
		input       string
		expected    BencodeDict
		expectError bool
	}{
		// should work as expected for empty dict
		{
			input:       "de",
			expected:    BencodeDict{},
			expectError: false,
		},
		// should work as expected for single item dict
		{
			input:       "d1:ki5ee",
			expected:    BencodeDict{"k": 5},
			expectError: false,
		},
		// should work as expected for list of diff items
		{
			input:       "d1:ki5e1:s1:se",
			expected:    BencodeDict{"k": 5, "s": "s"},
			expectError: false,
		},
		// should work as expected for nested dicts
		{
			input:       "d1:dd1:s1:see",
			expected:    BencodeDict{"d": BencodeDict{"s": "s"}},
			expectError: false,
		},
		// should return an error if it's not a dict start
		{
			input:       "ve",
			expected:    nil,
			expectError: true,
		},
		// should return an error if dict does not end properly
		{
			input:       "d1:h1:hE",
			expected:    nil,
			expectError: true,
		},
		// should return an error if key is not string
		{
			input:       "di5ei5ee",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "dl1:hei5ee",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "ddei5ee",
			expected:    nil,
			expectError: true,
		},
	}

	for _, test := range tests {
		i := 0
		res, err := consumeDict(test.input, &i)

		if test.expectError && err == nil {
			t.Errorf("expected an error but got '%s' as input %d", test.input, i)
		}

		if !test.expectError {
			if err != nil {
				t.Errorf("was not expecting an error got '%s' instead", err)
			}

			if !reflect.DeepEqual(res, test.expected) {
				t.Errorf("inputted '%s', expected ( %+v ) got ( %+v )", test.input, test.expected, res)
			}
		}

	}
}

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
		{
			input:       "lded1:h1:hee",
			expected:    []any{BencodeDict{}, BencodeDict{"h": "h"}},
			expectError: false,
		},
		// should return an error if an item in the list is wrong
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
