package adapter

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = make(map[string]Adapter)
)

// Register registers an adapter. It panics if the adapter is nil or already registered.
func Register(a Adapter) {
	mu.Lock()
	defer mu.Unlock()
	if a == nil {
		panic("adapter: Register adapter is nil")
	}
	name := a.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("adapter: Register called twice for adapter %s", name))
	}
	registry[name] = a
}

// Get returns the adapter with the given name, or nil if not found.
func Get(name string) Adapter {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// List returns a sorted list of registered adapter names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
