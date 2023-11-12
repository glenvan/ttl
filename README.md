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
  - Addition of `ttl.Map.Range()` in place of the `All()` method
    - Modelled after [`sync.Map.Range()`](https://pkg.go.dev/sync@go1.21.4#Map.Range)
    - Safe for concurrent access
- Replace internal `time.Tick()` with a `time.Ticker` to prevent leakage

## License

This project is licensed under the terms of [the MIT License](./LICENSE). It derives from
previous work, also licensed under the terms of [the MIT License](./LICENSE.orig.txt).

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
	maxTTL := 300 * time.Millisecond        // a key's time to live
	startSize := 3                          // initial number of items in map
	pruneInterval := 100 * time.Millisecond // prune expired items each time pruneInterval elapses
	refreshLastAccessOnGet := true          // update item's 'lastAccessTime' on ttl.Map.Load()

	// Any comparable data type such as int, uint64, pointers and struct types (if all field
	// types are comparable) can be used as the key type
	t := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
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
```

Output:

```bash
$ go run small.go

ttl.Map length: 2
[  hello] 'world'
Sleeping 400ms, items should be expired and removed afterward
[  hello] '' (exists: false)
[goodbye] '' (exists: false)
ttl.Map length: 0
```

## API

See the [package documentation](https://pkg.go.dev/github.com/glenvan/ttl).

## Acknowledgments

As mentioned, this is a fork of the awesome
[github.com/jftuga/TtlMap](https://github.com/jftuga/TtlMap) package. All ideas in this derivative
package flowed from that one.

### Original Package Acknowledgements

- Adopted from: [Map with TTL option in Go](https://stackoverflow.com/a/25487392/452281)
  - Answer created by: [OneOfOne](https://stackoverflow.com/users/145587/oneofone)
- [/u/skeeto](https://old.reddit.com/user/skeeto): suggestions for the `New` function
- `@zhayes` on the Go Discord: helping the original author with Go Generics

## Disclosure Notification

This program was completely developed on my own personal time, for my own personal benefit, and on
my personally owned equipment.
