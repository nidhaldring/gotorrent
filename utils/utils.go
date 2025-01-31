package utils

import (
	"errors"
	"fmt"
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
// Note: this assume that naming conversion between the map & struct
// follows the "transformName" algo.
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
			fmt.Printf("[Warning]: found %s in map, transformed it into %s but could not find in struct", k, fieldName)
			continue
		}

		nestedMap, ok := v.(map[string]any)
		if ok {
			err := MapToStruct(nestedMap, structValue.FieldByName(field.Name).Addr().Interface())
			if err != nil {
				return err
			}
		} else {
			structValue.FieldByName(field.Name).Set(reflect.ValueOf(v))
		}
	}

	return nil
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
