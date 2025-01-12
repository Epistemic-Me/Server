package db

import (
	"fmt"
	"reflect"
	"strings"

	"epistemic-me-core/pb/models"
)

// TestStruct is used for testing purposes
type TestStruct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// typeRegistry maps type names to their reflect.Type
var typeRegistry = map[string]reflect.Type{
	"string":        reflect.TypeOf(""),
	"int":           reflect.TypeOf(0),
	"bool":          reflect.TypeOf(true),
	"TestStruct":    reflect.TypeOf(TestStruct{}),
	"db.TestStruct": reflect.TypeOf(TestStruct{}),
}

// RegisterType registers a type with the type registry
func RegisterType(v interface{}) {
	t := reflect.TypeOf(v)
	typeRegistry[t.String()] = t
	// Also register the pointer type
	typeRegistry["*"+t.String()] = reflect.PointerTo(t)
	// Register with package name
	typeRegistry[t.PkgPath()+"."+t.Name()] = t
	typeRegistry["*"+t.PkgPath()+"."+t.Name()] = reflect.PointerTo(t)
}

// Helper function to get reflect.Type from type name
func getTypeFromName(name string) (reflect.Type, error) {
	// Check if it's a pointer type
	isPtr := strings.HasPrefix(name, "*")
	if isPtr {
		name = name[1:] // Remove the "*" prefix
	}

	// Try to find the type with and without package name
	t, ok := typeRegistry[name]
	if !ok {
		// If not found, try splitting by dot and use the last part
		parts := strings.Split(name, ".")
		t, ok = typeRegistry[parts[len(parts)-1]]
	}
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", name)
	}

	if isPtr {
		return reflect.PointerTo(t), nil
	}
	return t, nil
}

// Initialize the type registry with your known types
func init() {
	// Register models.Developer
	RegisterType(models.Developer{})
	// Register models.Belief
	RegisterType(models.Belief{})
	// Register models.BeliefSystem
	RegisterType(models.BeliefSystem{})
	// Register models.Dialectic
	RegisterType(models.Dialectic{})
	RegisterType(models.SelfModel{})
	RegisterType(models.User{})
	RegisterType(models.Philosophy{})
	// Register TestStruct
	RegisterType(TestStruct{})
}
