package module

import (
	"fmt"
	"sort"
)

var registry = map[string]InitFunc{}

// Config is the conduit for providing initialization information to a module
type Config map[string]interface{}

// InitFunc is a factory function that returns a module
type InitFunc func(Config) (Patcher, error)

// Lookup retrieves a module's initialization function from the Registry
func Lookup(name string) (InitFunc, error) {
	c, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("module type not registered: %s", name)
	}
	return c, nil
}

// Register registers a module under a specified name. This is then exposed as a constructor in the `eolian.synth`
// package at the Lua layer.
func Register(name string, fn func(Config) (Patcher, error)) {
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("%s already registered as a module", name))
	}
	registry[name] = fn
}

// RegisteredTypes returns a list of all registered module types
func RegisteredTypes() []string {
	types := []string{}
	for k := range registry {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}
