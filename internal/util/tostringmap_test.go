package util

import (
	"testing"
)

func TestToStringMap_MapStringAny(t *testing.T) {
	input := map[string]any{"key": "val"}
	got := ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_MapStringString(t *testing.T) {
	input := map[string]string{"key": "val"}
	got := ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_JSONBytes(t *testing.T) {
	input := []byte(`{"key":"val"}`)
	got := ToStringMap(input)
	if got["key"] != "val" {
		t.Errorf("expected val, got %v", got["key"])
	}
}

func TestToStringMap_EmptyBytes(t *testing.T) {
	input := []byte{}
	got := ToStringMap(input)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestToStringMap_InvalidBytes(t *testing.T) {
	input := []byte(`not json`)
	got := ToStringMap(input)
	if got != nil {
		t.Errorf("expected nil for invalid json, got %v", got)
	}
}

func TestToStringMap_Struct(t *testing.T) {
	input := struct {
		Name string `json:"name"`
	}{Name: "test"}
	got := ToStringMap(input)
	if got["name"] != "test" {
		t.Errorf("expected test, got %v", got["name"])
	}
}

func TestToStringMap_NonMapType(t *testing.T) {
	got := ToStringMap(42)
	if got != nil {
		t.Errorf("expected nil for int, got %v", got)
	}
}

func TestToStringMap_Nil(t *testing.T) {
	got := ToStringMap(nil)
	if got != nil {
		t.Error("expected nil for nil input")
	}
}
