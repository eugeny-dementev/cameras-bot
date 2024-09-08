package main

import (
	"sync"
)

// A basic handler State to share state across executions.
// Note: This is a very simple layout which uses a shared mutex.
// It is all in-memory, and so will not persist data across restarts.
type State struct {
	// Use a mutex to avoid concurrency issues.
	// If you use multiple maps, you may want to use a new mutex for each one.
	rwMux sync.RWMutex

	// We use a double map to:
	// - map once for the user id
	// - map a second time for the keys a user can have
	// The second map has values of type "any" so anything can be stored in them, for the purpose of this example.
	// This could be improved by using a struct with typed fields, though this would need some additional handling to
	// ensure concurrent safety.
	userData map[int64]map[string]any

	// This struct could also contain:
	// - pointers to database connections
	// - pointers cache connections
	// - localised strings
	// - helper methods for retrieving/caching chat settings
}

func (c *State) Setup() {
	if c.userData == nil {
		c.userData = map[int64]map[string]any{}
	}
}

func (c *State) Get(userId int64, key string) (any, bool) {
	c.rwMux.RLock()
	defer c.rwMux.RUnlock()

	if c.userData == nil {
		return nil, false
	}

	userData, ok := c.userData[userId]
	if !ok {
		return nil, false
	}

	v, ok := userData[key]
	return v, ok
}

func (c *State) Set(userId int64, key string, val any) {
	c.rwMux.Lock()
	defer c.rwMux.Unlock()

	_, ok := c.userData[userId]
	if !ok {
		c.userData[userId] = map[string]any{}
	}
	c.userData[userId][key] = val
}
