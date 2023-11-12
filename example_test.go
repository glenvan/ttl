package ttl_test

import (
	"fmt"
	"time"

	"github.com/glenvan/ttl"
)

func ExampleMap() {
	defaultTTL := 300 * time.Millisecond    // the default lifetime for a key/value pair
	startSize := 3                          // initial number of items in map
	pruneInterval := 100 * time.Millisecond // prune expired items each time pruneInterval elapses
	refreshOnLoad := true                   // update item's 'lastAccessTime' on ttl.Map.Load()

	// Any comparable data type such as int, uint64, pointers and struct types (if all field
	// types are comparable) can be used as the key type
	t := ttl.NewMap[string, string](defaultTTL, startSize, pruneInterval, refreshOnLoad)
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

	sleepTime := defaultTTL + pruneInterval
	fmt.Printf("Sleeping %s, items should be expired and removed afterward\n", sleepTime)

	time.Sleep(sleepTime)

	v, ok := t.Load("hello")
	fmt.Printf("[%7s] '%v' (exists: %t)\n", "hello", v, ok)

	v, ok = t.Load("goodbye")
	fmt.Printf("[%7s] '%v' (exists: %t)\n", "goodbye", v, ok)

	fmt.Printf("ttl.Map length: %d\n", t.Length())
	// Output:
	// ttl.Map length: 2
	// [  hello] 'world'
	// Sleeping 400ms, items should be expired and removed afterward
	// [  hello] '' (exists: false)
	// [goodbye] '' (exists: false)
	// ttl.Map length: 0
}

func ExampleMap_Load() {
	tm := ttl.NewMap[string, string](30*time.Second, 0, 2*time.Second, true)
	defer tm.Close()

	tm.Store("hello", "world")

	value, ok := tm.Load("hello")
	if ok {
		fmt.Println(value)
	}
	// Output:
	// world
}

func ExampleMap_Range() {
	tm := ttl.NewMap[string, string](30*time.Second, 0, 2*time.Second, true)
	defer tm.Close()

	tm.Store("hello", "world")
	tm.Store("goodbye", "universe")

	fmt.Printf("Length before: %d\n", tm.Length())

	tm.Range(func(key string, val string) bool {
		if key == "goodbye" {
			// defer deletion in the original Map using a goroutine
			go func() {
				tm.Delete(key)
			}()

			return false // break
		}

		return true // continue
	})

	// Give the goroutine some time to complete
	time.Sleep(20 * time.Millisecond)

	fmt.Printf("Length after: %d\n", tm.Length())
	// Output:
	// Length before: 2
	// Length after: 1
}

func ExampleMap_DeleteFunc() {
	tm := ttl.NewMap[string, int](30*time.Second, 0, 2*time.Second, true)
	defer tm.Close()

	tm.Store("zero", 0)
	tm.Store("one", 1)
	tm.Store("two", 2)

	// Delete all even keys
	tm.DeleteFunc(func(key string, val int) bool {
		return val%2 == 0
	})

	tm.Range(func(key string, val int) bool {
		fmt.Printf("%s: %d\n", key, val)
		return true
	})
	// Output:
	// one: 1
}
