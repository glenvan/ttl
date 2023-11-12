package main

import (
	"fmt"
	"time"

	"github.com/glenvan/ttl"
)

func main() {
	maxTTL := time.Duration(time.Second * 4)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	t := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer t.Close()

	// populate the ttl.Map
	t.Store("string", "a b c")
	t.Store("int", "3")
	t.Store("float", "4.4")
	t.Store("int_array", "1, 2, 3")
	t.Store("bool", "false")
	t.Store("rune", "{")
	t.Store("byte", "0x7b")
	t.Store("uint64", "123456789")

	fmt.Println()
	fmt.Println("ttl.Map length:", t.Length())

	// display all items in ttl.Map
	fmt.Println()
	fmt.Println("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
	t.Range(func(key string, value string) bool {
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
	sleepTime := maxTTL + pruneInterval
	fmt.Println()
	fmt.Printf("Sleeping %v seconds, items should be removed after this time, except for the '%v' key\n", sleepTime, dontExpireKey)
	fmt.Println()
	time.Sleep(sleepTime)

	// these items have expired and therefore should be nil, except for 'dontExpireKey'
	v, ok := t.Load("string")
	fmt.Printf("[%9s] %v (%t)\n", "string", v, ok)
	v, ok = t.Load("int")
	fmt.Printf("[%9s] %v (%t)\n", "int", v, ok)
	v, ok = t.Load("float")
	fmt.Printf("[%9s] %v (%t)\n", "float", v, ok)
	v, ok = t.Load("int_array")
	fmt.Printf("[%9s] %v (%t)\n", "int_array", v, ok)
	v, ok = t.Load("bool")
	fmt.Printf("[%9s] %v (%t)\n", "bool", v, ok)
	v, ok = t.Load("rune")
	fmt.Printf("[%9s] %v (%t)\n", "rune", v, ok)
	v, ok = t.Load("byte")
	fmt.Printf("[%9s] %v (%t)\n", "byte", v, ok)
	v, ok = t.Load("uint64")
	fmt.Printf("[%9s] %v (%t)\n", "uint64", v, ok)

	// sanity check, this key should exist
	fmt.Println()
	if v, ok := t.Load("int"); ok {
		fmt.Printf("[int] is %s", v)
	}
	fmt.Println("ttl.Map length:", t.Length(), " (should equal 1)")
	fmt.Println()

	fmt.Println()
	fmt.Printf("Manually deleting '%v' key; should be successful\n", "int")
	t.Delete("int")
	_, ok = t.Load("int")
	fmt.Printf("    successful? %t\n", !ok)
	fmt.Println("ttl.Map length:", t.Length(), " (should equal 0)")
	fmt.Println()

	fmt.Println("Adding 2 items and then running Clear()")
	t.Store("string", "a b c")
	t.Store("int", "3")
	fmt.Println("ttl.Map length:", t.Length())

	fmt.Println()
	fmt.Println("Running Clear()")
	t.Clear()
	fmt.Println("ttl.Map length:", t.Length())
	fmt.Println()
}
