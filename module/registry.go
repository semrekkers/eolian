package module

import (
	"fmt"
	"sort"
)

var Registry = map[string]InitFunc{}

// Config is the conduit for providing initialization information to a module
type Config map[string]interface{}

// InitFunc is a factory function that returns a module
type InitFunc func(Config) (Patcher, error)

// Lookup retrieves a module's initialization function from the Registry
func Lookup(name string) (InitFunc, error) {
	c, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("module type not registered: %s", name)
	}
	return c, nil
}

// Register registeres an input under a specified name
func Register(name string, fn func(Config) (Patcher, error)) {
	if _, ok := Registry[name]; ok {
		panic(fmt.Sprintf("%s already registered as a module", name))
	}
	Registry[name] = fn
}

// ReggisteredTypes returns a list of all registered module types
func RegisteredTypes() []string {
	types := []string{}
	for k := range Registry {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}
