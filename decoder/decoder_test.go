package decoder

import (
	// "os"
	"reflect"
	"testing"
)

// func TestParseBenCode(t *testing.T) {
// 	// @TODO: write better tests
// 	expectedResult := &TorrentFile{
// 		Announce: "udp://tracker.opentrackr.org:1337/announce",
// 		AnnounceList: []any{
// 			[]any{
// 				"udp://tracker.opentrackr.org:1337/announce",
// 			},
// 			[]any{
// 				"udp://opentracker.io:6969/announce",
// 			},
// 			[]any{
// 				"udp://tracker.dler.org:6969/announce",
// 			},
// 			[]any{
// 				"udp://open.publictracker.xyz:6969/announce",
// 			},
// 			[]any{
// 				"udp://tracker.dler.com:6969/announce",
// 			},
// 			[]any{
// 				"udp://opentracker.io:6969/announce",
// 			},
// 			[]any{
// 				"udp://tracker.opentrackr.org:1337/",
// 			},
// 			[]any{
// 				"http://tracker.bt4g.com:2095/announce",
// 			},
// 			[]any{
// 				"udp://amigacity.xyz:6969/announce",
// 			},
// 			[]any{
// 				"udp://tracker.torrent.eu.org:451/announce",
// 			},
// 			[]any{
// 				"udp://retracker.lanta.me:2710/announce",
// 			},
// 			[]any{
// 				"udp://tracker.0x7c0.com:6969/announce",
// 			},
// 			[]any{
// 				"udp://ttk2.nbaonlineservice.com:6969/announce",
// 			},
// 			[]any{
// 				"udp://tracker.torrent.eu.org:451/announce",
// 			},
// 			[]any{
// 				"udp://seedpeer.net:6969/announce",
// 			},
// 		},
// 		CreatedBy:    "uTorrent/3.6",
// 		CreationDate: 1737709167,
// 		Encoding:     "UTF-8",
// 		Info: TorrentInfo{
// 			Length:      1315136554,
// 			Name:        "Star Trek  Section 31 2025 1080p WEB-DL HEVC x265 5.1 BONE.mkv",
// 			PieceLength: 2097152,
//       // @TODO: enable Pieces
//       // Pieces:``,
// 		},
// 	}
// 	testFilePath := "./files/test.torrent"

// 	b, err := os.ReadFile(testFilePath)
// 	if err != nil {
// 		t.Fatalf("Test file %s was not found!", testFilePath)
// 	}

// 	content := string(b)
// 	parsed, err := DecodeTorrentFile(content)
// 	if err != nil {
// 		t.Fatalf("Expected ParseBencode to return result got error %s instead", err)
// 	}

// 	if parsed == nil {
// 		t.Fatal("Expected result to be different to nil")
// 	}

// 	if parsed != nil && !reflect.DeepEqual(expectedResult, parsed) {
// 		t.Fatalf("Expected results to be equal inputted file %s represented as %+v got %+v instead", testFilePath, *expectedResult, *parsed)
// 	}
// }

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
