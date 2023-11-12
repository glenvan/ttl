package main

import (
	"context"
	"fmt"
	"time"

	"github.com/glenvan/ttl"
)

func main() {
	maxTTL := 300 * time.Millisecond        // a key's time to live
	startSize := 3                          // initial number of items in map
	pruneInterval := 100 * time.Millisecond // prune expired items each time pruneInterval elapses
	refreshOnStore := true                  // update item's 'lastAccessTime' on ttl.Map.Load()

	// Any comparable data type such as int, uint64, pointers and struct types (if all field
	// types are comparable) can be used as the key type
	t := ttl.NewMap[string, string](
		context.Background(),
		maxTTL,
		startSize,
		pruneInterval,
		refreshOnStore)
	defer t.Close()

	// Populate the ttl.Map
	t.Store("hello", "world")
	t.Store("goodbye", "universe")

	fmt.Printf("ttl.Map length: %d\n", t.Length())

	t.Delete("goodbye")

	// Display all items in ttl.Map
	t.Range(func(key string, value string) bool {
		fmt.Printf("[%7s] '%v'\n", key, value)
		return true
	})

	sleepTime := maxTTL + pruneInterval
	fmt.Printf("Sleeping %s, items should be expired and removed afterward\n", sleepTime)

	time.Sleep(sleepTime)

	v, ok := t.Load("hello")
	fmt.Printf("[%7s] '%v' (exists: %t)\n", "hello", v, ok)

	v, ok = t.Load("goodbye")
	fmt.Printf("[%7s] '%v' (exists: %t)\n", "goodbye", v, ok)

	fmt.Printf("ttl.Map length: %d\n", t.Length())
}
