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
// Note: this makes all map fields public
func MapToStruct(m map[string]any, s any) error {
	vr := reflect.ValueOf(s)
	if vr.Kind() != reflect.Pointer && vr.IsValid() && vr.Elem().Kind() != reflect.Struct {
		return errors.New("Given value is not a struct")
	}

	structValue := vr.Elem()
	structType := structValue.Type()
	for i := range structValue.NumField() {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		var v any

		// Either find the field name as it's or
		// try to find it "uncapitalized" in the map
		tmp, ok := m[field.Name]
		if ok {
			v = tmp
		} else {
			// try to "uncapitalized" field name
			capitalizedName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
			tmp, ok := m[capitalizedName]
			if !ok {
				return errors.New(fmt.Sprintf("did not found %s or %s in map", field.Name, capitalizedName))
			}
			v = tmp
		}

		nestedMap, ok := v.(map[string]any)
		if ok {
			err := MapToStruct(nestedMap, structValue.FieldByName(field.Name).Addr().Interface())
			if err != nil {
				return err
			}
		} else {
			structValue.Field(i).Set(reflect.ValueOf(v))
		}
	}

	return nil
}
