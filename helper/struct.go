package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func MapToStruct[T any](data map[string]interface{}) (*T, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal map to json: %w", err)
	}

	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal json to struct: %w", err)
	}

	return &result, nil
}

func MergeStruct[T any](base T, override T) T {
	baseValue := reflect.ValueOf(&base).Elem()
	overrideValue := reflect.ValueOf(override)

	for i := 0; i < baseValue.NumField(); i++ {
		field := baseValue.Field(i)
		overrideField := overrideValue.Field(i)

		if !overrideField.IsZero() {
			if overrideField.Kind() == reflect.Struct {
				// Get the concrete type of the nested struct
				baseField := field.Interface()

				// Create a new merged value using reflection
				mergedValue := reflect.ValueOf(baseField)
				merged := reflect.New(mergedValue.Type()).Elem()
				merged.Set(mergedValue)

				// Iterate through the nested struct fields
				for j := 0; j < overrideField.NumField(); j++ {
					if !overrideField.Field(j).IsZero() {
						merged.Field(j).Set(overrideField.Field(j))
					}
				}

				field.Set(merged)
			} else {
				field.Set(overrideField)
			}
		}
	}

	return base
}
