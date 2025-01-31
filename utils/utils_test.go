package utils

import (
	"reflect"
	"testing"
)

func TestStructToMap(t *testing.T) {
	tests := []struct {
		input    any // struct
		expected map[string]any
	}{
		// note: TestStructToMap only works on public fields
		// this is a test for that
		{
			input: struct {
				h int
				L string
			}{h: 1, L: "l"},
			expected: map[string]any{"L": "l"},
		},
		{
			input:    struct{}{},
			expected: map[string]any{},
		},
		{
			input:    struct{ F struct{ X int } }{F: struct{ X int }{X: 5}},
			expected: map[string]any{"F": map[string]any{"X": 5}},
		},
	}

	for _, test := range tests {
		res, err := StructToMap(test.input)

		if err != nil {
			t.Fatalf("expected no error got %s instead", err)
		}

		if !reflect.DeepEqual(res, test.expected) {
			t.Errorf("input = %+v and expected %+v got %+v", test.input, test.expected, res)
		}
	}
}

func TestMapToStruct(t *testing.T) {
	// Note: res type & expected types are anoynmous so that
	// private fields are not accessed by MapToStruct
	// simulating real world scenario
	input := map[string]any{"f": map[string]any{"x": 5}}
	var res struct{ F struct{ X int } }

	err := MapToStruct(input, &res)

	if err != nil {
		t.Fatalf("expected no error got %s instead", err)
	}

	expected := struct{ F struct{ X int } }{F: struct{ X int }{X: 5}}
	if !reflect.DeepEqual(res, expected) {
		t.Errorf("input = %+v and expected %+v got %+v", input, expected, res)
	}
}

func TestTransformName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    "a",
			expected: "A",
		},
		{
			input:    "normal",
			expected: "Normal",
		},
		{
			input:    "kebab-case",
			expected: "KebabCase",
		},
		{
			input:    "with space",
			expected: "WithSpace",
		},
	}

	for _, test := range tests {
		res := transformName(test.input)
		if test.expected != res {
			t.Fatalf("Expected '%s' for input '%s' got '%s' instead", test.expected, test.input, res)
		}
	}
}
