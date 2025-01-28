package encoder

import (
	"errors"
	"fmt"
	"gotorrent/decoder"
	"reflect"
)

func Encode(v any) (string, error) {
	switch val := v.(type) {
	case int:
		return encodeInt(val), nil
	case string:
		return encodeString(val), nil
	case []any:
		return encodeList(val), nil
	case decoder.BencodeDict:
		return encodeDict(val), nil
	}

	if reflect.TypeOf(v).Kind() == reflect.Struct {

	}

	return "", errors.New("given type is not supported")
}

func encodeDict(dict decoder.BencodeDict) string {
	encodedStr := ""
	for k, v := range dict {
		switch val := v.(type) {
		case int:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeInt(val))
		case string:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeString(val))
		case []any:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeList(val))
		case decoder.BencodeDict:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeDict(val))
		}
	}

	return "d" + encodedStr + "e"
}

func encodeList(list []any) string {
	encodedStr := ""
	for _, v := range list {
		switch val := v.(type) {
		case int:
			encodedStr += encodeInt(val)
		case string:
			encodedStr += encodeString(val)
		case []any:
			encodedStr += encodeList(val)
		case decoder.BencodeDict:
			encodedStr += encodeDict(val)
		}
	}

	return "l" + encodedStr + "e"
}

func encodeString(s string) string {
	return fmt.Sprintf("%d:%s", len(s), s)
}

func encodeInt(n int) string {
	return fmt.Sprintf("i%de", n)
}

// this implementation is simple and optimized only for our use cases
// note: this panics if struct has private fields
// note: it returns capitalized map keys
func structToMap(s any) (map[string]any, error) {
	typ := reflect.TypeOf(s)
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("Given value is not a struct")
	}

	val := reflect.ValueOf(s)
	m := make(map[string]any)
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type.Kind() == reflect.Struct {
			nestedMap, err := structToMap(val.Field(i).Interface())
			if err != nil {
				return nil, err
			}

			m[field.Name] = nestedMap
		} else {
			m[field.Name] = val.Field(i).Interface()
		}
	}

	return m, nil
}
