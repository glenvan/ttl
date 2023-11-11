package main

import (
	"fmt"
	"time"

	"github.com/glenvan/ttl"
)

func main() {
	maxTTL := time.Duration(time.Second * 4)        // a key's time to live in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's 'lastAccessTime' on a .Get()

	// any comparable data type such as int, uint64, pointers and struct types (if all field types are comparable)
	// can be used as the key type, not just string
	t := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer t.Close()

	// populate the TtlMap
	t.Store("myString", "a b c")
	t.Store("int_array", "1, 2, 3")
	fmt.Println("TtlMap length:", t.Length())

	// display all items in TtlMap
	t.Range(func(key string, value string) bool {
		fmt.Printf("[%9s] %v\n", key, value)
		return true
	})

	fmt.Println()

	sleepTime := maxTTL + pruneInterval
	fmt.Printf("Sleeping %v seconds, items should be 'nil' after this time\n", sleepTime)
	time.Sleep(sleepTime)
	v, ok := t.Load("myString")
	fmt.Printf("[%9s] %v (exists: %t)\n", "myString", v, ok)
	v, ok = t.Load("int_array")
	fmt.Printf("[%9s] %v (exists: %t)\n", "int_array", v, ok)
	fmt.Println("TtlMap length:", t.Length())
}
