package parser

import (
	"reflect"
	"testing"
)

type testCase struct {
	bencode      string
	expectedDict BencodeDict
}

func TestParseBenCode(t *testing.T) {
	tests := []testCase{
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

func TestParseBenCodeWithEmptyString(t *testing.T) {
	res, err := ParseBencode("")
	if err == nil {
		t.Errorf("failed to parse empty string expected an error got %+v instead", res)
	}
}
