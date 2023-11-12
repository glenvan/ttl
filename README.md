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

```go
type Map[K comparable, V any] struct {
	// Has unexported fields.
}
```

`Map` is a "time-to-live" map such that after a given amount of time, items in
the map are deleted. `Map` is safe for concurrent use.

When a `Map.Load` or `Map.Store` occurs, the lastAccess time is set to the
current time. Therefore, only items that are not called by `Map.Load` or
`Map.Store` will be deleted after the TTL expires.

`Map.LoadPassive` can be used in which case the lastAccess time will *not* be
updated.

Adapted from:
[https://stackoverflow.com/a/25487392/452281](https://stackoverflow.com/a/25487392/452281)

```go
func NewMap[K comparable, V any](
	maxTTL time.Duration,
	length int,
	pruneInterval time.Duration,
	refreshLastAccessOnGet bool,
) (m *Map[K, V])
```

`NewMap` returns a new Map with items expiring according to the maxTTL
specified if they have not been accessed within that duration. Access
refresh can be overridden so that items expire after the TTL whether they
have been accessed or not.

```go
func (m *Map[K, V]) Clear()
```

`Clear` will remove all key/value pairs from the `Map`. `Clear` is safe for
concurrent use.

```go
func (m *Map[K, V]) Close()
```

`Close` will terminate TTL pruning of the `Map`. If `Close` is not called on a
`Map` after its no longer needed, the `Map` will leak.

```go
func (m *Map[K, V]) Delete(key K)
```

`Delete` will remove a key and its value from the `Map`. `Delete` is safe for
concurrent use.

```go
func (m *Map[K, V]) Length() int
```

`Length` returns the current length of the `Map`'s internal map. `Length` is safe
for concurrent use.

```go
func (m *Map[K, V]) Load(key K) (value V, ok bool)
```

`Load` will retrieve a value from the `Map`, as well as a bool indicating
whether the key was found. If the item was not found the value returned is
undefined. `Load` is safe for concurrent use.

```go
func (m *Map[K, V]) LoadPassive(key K) (value V, ok bool)
```

`LoadPassive` will retrieve a value from the `Map` (without updating that
value's time to live), as well as a bool indicating whether the key
was found. If the item was not found the value returned is undefined.
`LoadPassive` is safe for concurrent use.

```go
func (m *Map[K, V]) Range(f func(key K, value V) bool)
```

`Range` calls f sequentially for each key and value present in the Map.
If f returns false, `Range` stops the iteration.

`Range` is safe for concurrent use and supports modifying the value (assuming
it's a reference type like a slice, map, or a pointer) within the range
function. However, this requires a write lock on the `Map` – so you are
not able to perform `Map.Delete` or `Map.Store` operations on the original
`Map` directly within the range func, as that would cause a panic. Even an
accessor like `Map.Load` or `Map.LoadPassive` would lock indefinitely.

If you need to perform operations on the original `Map`, do so in a new
goroutine from within the range func – effectively deferring the operation
until the `Range` completes.

```go
func (m *Map[K, V]) Store(key K, value V)
```

`Store` will insert a value into the Map. `Store` is safe for concurrent use.

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
