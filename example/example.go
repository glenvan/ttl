package main

import (
	"fmt"
	"time"

	"github.com/glenvan/ttl"
)

func main() {
	defaultTTL := time.Duration(time.Second * 4)    // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshOnLoad := true                           // update item's lastAccessTime on a .Get()

	t := ttl.NewMap[string, any](defaultTTL, startSize, pruneInterval, refreshOnLoad)
	defer t.Close()

	// populate the ttl.Map
	t.Store("string", "a b c")
	t.Store("int", 3)
	t.Store("float", 4.4)
	t.Store("int slice", []int{1, 2, 3})
	t.Store("bool", false)
	t.Store("rune", '}')
	t.Store("byte", byte(0x7b))
	t.Store("uint64", uint64(123456789))

	fmt.Println()
	fmt.Println("ttl.Map length:", t.Length())

	// display all items in ttl.Map
	fmt.Println()
	fmt.Println("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
	t.Range(func(key string, value any) bool {
		fmt.Printf("[%9s] %v\n", key, value)
		return true
	})
	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
	fmt.Println()

	// by executing Get(), the 'dontExpireKey' lastAccessTime will be updated
	// therefore, this item will not expire
	dontExpireKey := "float"
	go func() {
		for range time.Tick(time.Second) {
			t.Load(dontExpireKey)
		}
	}()

	// ttl.Map has an expiration time, wait until this amount of time passes
	sleepTime := defaultTTL + 2*pruneInterval
	fmt.Println()
	fmt.Printf(
		"Sleeping %v, items should be removed after this time, except for the '%v' key\n",
		sleepTime,
		dontExpireKey)
	fmt.Println()
	time.Sleep(sleepTime)

	// these items have expired and therefore should be nil, except for 'dontExpireKey'
	keys := []string{"string", "int", "float", "int slice", "bool", "rune", "byte", "uint64"}
	for _, key := range keys {
		v, ok := t.Load(key)
		fmt.Printf("[%9s] %v (%t)\n", key, v, ok)
	}

	// sanity check, this key should exist
	fmt.Println()
	if v, ok := t.Load("int"); ok {
		fmt.Printf("[int] is %s", v)
	}
	fmt.Println("ttl.Map length:", t.Length(), " (should equal 1)")
	fmt.Println()

	fmt.Println()
	fmt.Printf("Manually deleting '%v' key; should be successful\n", dontExpireKey)
	t.Delete(dontExpireKey)
	_, ok := t.Load(dontExpireKey)
	fmt.Printf("    successful? %t\n", !ok)
	fmt.Println("ttl.Map length:", t.Length(), " (should equal 0)")
	fmt.Println()

	fmt.Println("Adding 2 items and then running Clear()")
	t.Store("string", "a b c")
	t.Store("int", 3)
	fmt.Println("ttl.Map length:", t.Length())

	fmt.Println()
	fmt.Println("Running Clear()")
	t.Clear()
	fmt.Println("ttl.Map length:", t.Length())
	fmt.Println()
}
