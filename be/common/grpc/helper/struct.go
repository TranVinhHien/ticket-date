package grpcHelper

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertMapToStructPB converts a map[string]any to *structpb.Struct.
// Returns an empty structpb.Struct if the input is nil.
func ConvertMapToStructPB(data map[string]any) (*structpb.Struct, error) {
	if data == nil {
		return &structpb.Struct{Fields: make(map[string]*structpb.Value)}, nil
	}

	return structpb.NewStruct(data)
}

// ConvertStructPBToMap converts *structpb.Struct to map[string]any.
// Returns an empty map if structPB is nil.
func ConvertStructPBToMap(structPB *structpb.Struct) map[string]any {
	if structPB == nil || structPB.Fields == nil {
		return make(map[string]any)
	}

	result := make(map[string]any, len(structPB.Fields))
	for key, value := range structPB.Fields {
		result[key] = value.AsInterface()
	}
	return result
}

// ConvertAnyToValuePB wraps any supported primitive, slice, or map into *structpb.Value.
func ConvertAnyToValuePB(v any) (*structpb.Value, error) {
	return structpb.NewValue(v)
}
