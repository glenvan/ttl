# ttl

## Status

Pre-release, use with caution.

## Introduction

`ttl` is golang package that implements *time-to-live* container types such that after a given
amount of time, items in the map are deleted.

- The map key can be any [comparable](https://go.dev/ref/spec#Comparison_operators) data type, via
  Generics.
- Any data type can be used as a map value, via Generics.

This is a fork of the awesome [github.com/jftuga/TtlMap](https://github.com/jftuga/TtlMap) with a
few enhancements and creature-comforts:

- `Map` is a generic, homogeneous map type
  - Meaning that both key and value are determined by Generics
  - Using `any` or `interface{}` as the value type will effectively emulate the original source
    package
- The package name is simply `ttl`, in case other TTL-enabled types seem like a good idea
  - For example: a slice implementation
- The syntax is a little more idiomatic
- Methods have been renamed to be more familiar to Go standard library users
  - `Load()` and `Store()` instead of `Get()` and `Set()`
- Code is a little safer for concurrent use (at the time of the fork) and more performant in that
  use case
  - Use of `sync.RWLock` so that read-heavy applications block less
  - Use of `atomic.Int64` for the timestamp so that it may be updated without a write lock
  - Addition of `ttl.Map.Range()` as an alternative to the `All()` method
    - Modelled after [`sync.Map.Range()`](https://pkg.go.dev/sync@go1.21.4#Map.Range)
  - Addition of `ttl.Map.Eject()`, which produces a conventional `map[K]V` copy of the Map
    - Depending on the value type, this may be more convenient and possibly safe for concurrent use
      under the right circumstances
- Replace internal `time.Tick()` with a `time.Ticker` to prevent leakage

## Example

[Full example using many data types](example/example.go)

Small example:

```go
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
```

Output:

```bash
$ go run small.go

TtlMap length: 2
[ myString] a b c
[int_array] [1 2 3]

Sleeping 5 seconds, items should be 'nil' after this time
[ myString] <nil>
[int_array] <nil>
TtlMap length: 0
```

## API functions

- `New`: initialize a `TtlMap`
- `Close`: this stops the goroutine that checks for expired items; use with `defer`
- `Len`: return the number of items in the map
- `Put`: add a key/value
- `Get`: get the current value of the given key; return `nil` if the key is not found in the map
- `GetNoUpdate`: same as `Get`, but do not update the `lastAccess` expiration time
  - This ignores the `refreshLastAccessOnGet` parameter
- `All`: returns a *copy* of all items in the map
- `Delete`: delete an item; return `true` if the item was deleted, `false` if the item was not
  found in the map
- `Clear`: remove all items from the map

## Performance

- Searching for expired items runs in O(n) time, where n = number of items in the `TtlMap`.
  - This inefficiency can be somewhat mitigated by increasing the value of the `pruneInterval` time.
- In most cases you want `pruneInterval > maxTTL`; otherwise expired items will stay in the map
  longer than expected.

## Acknowledgments

- Adopted from: [Map with TTL option in Go](https://stackoverflow.com/a/25487392/452281)
  - Answer created by: [OneOfOne](https://stackoverflow.com/users/145587/oneofone)
- [/u/skeeto](https://old.reddit.com/user/skeeto): suggestions for the `New` function
- `@zhayes` on the Go Discord: helping me with Go Generics

## Disclosure Notification

This program was completely developed on my own personal time, for my own personal benefit, and on
my personally owned equipment.
