package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"
)

// Note: This implementation is simple and optimized only for our use cases
// Note: this only transform public fields
func StructToMap(s any) (map[string]any, error) {
	typ := reflect.TypeOf(s)
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("Given value is not a struct")
	}

	val := reflect.ValueOf(s)
	m := make(map[string]any)
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			nestedMap, err := StructToMap(val.Field(i).Interface())
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

// Note: This implementation is simple and optimized only for our use cases
// Note: This assume that naming conversion between the map & struct
// Note: This also assume that []any can be safely converted to [][]string
// follows the "transformName" algo.
// "s" should be a pointer to struct we want to pouplate
// @TODO: handle the case where as is pointer to struct pointer,
// this for some reasons passes the first condition and causes the code to panic!
func MapToStruct(m map[string]any, s any) error {
	vr := reflect.ValueOf(s)
	if vr.Kind() != reflect.Pointer && vr.IsValid() && vr.Elem().Kind() != reflect.Struct {
		return errors.New("Given value is not a struct")
	}

	structValue := vr.Elem()
	structType := structValue.Type()
	for k, v := range m {
		fieldName := transformName(k)
		field, found := structType.FieldByName(fieldName)
		// @TODO: maybe add in the struct field tag that this can be ignored
		// for now i'm just going to report it and move on.
		if !found {
			fmt.Printf("[Warning]: found %s in map, transformed it into %s but could not find in struct\n", k, fieldName)
			continue
		}

		nestedMap, ok := v.(map[string]any)
		if ok {
			err := MapToStruct(nestedMap, structValue.FieldByName(field.Name).Addr().Interface())
			if err != nil {
				return err
			}
		} else {
			// Super hacky! but i'm going to assume that any []any is an actual [][]string
			// because that's all i need for bencode :)!
			// @TODO: maybe come fix this at some point :)!
			anyr, ok := v.([]any)
			if ok {
				res := make([][]string, 0)
				for _, nr := range anyr {
					arr, ok := nr.([]any)
					if !ok {
						return errors.New("Expected []any to unwrap to [][]string")
					}

					inner := make([]string, 0)
					for _, v := range arr {
						s, ok := v.(string)
						if !ok {
							return errors.New(fmt.Sprintf("Expected nested array elements to of type string got %+v instead", v))
						}
						inner = append(inner, s)
					}
					res = append(res, inner)
				}

				structValue.FieldByName(field.Name).Set(reflect.ValueOf(res))
			} else {
				structValue.FieldByName(field.Name).Set(reflect.ValueOf(v))
			}
		}
	}

	return nil
}

func ConnectToUDPURL(u *url.URL) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func transformName(name string) string {
	mapF := func(arr []string, f func(string) string) string {
		res := ""
		for _, v := range arr {
			res += f(v)
		}
		return res
	}
	capitalize := func(s string) string {
		if len(s) == 0 {
			return ""
		}

		return strings.ToUpper(string(s[0])) + s[1:]
	}

	return mapF(strings.Split(mapF(strings.Split(name, " "), capitalize), "-"), capitalize)

}
