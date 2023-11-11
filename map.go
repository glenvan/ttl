package ttl

import (
	"sync"
	"sync/atomic"
	"time"
)

// MapRUnlockFunc is a function type returned by ttl.All() to release the read-lock on the Map
type MapRUnlockFunc func()

// MapItem is an internal record for values stored in a Map. It is primarily an internal type used
// by Map, but can also be used when accessing items using Map.All(). MapItem.Value is not safe
// for concurrent use, unless it has been returned by a method like Map.All() that holds an
// extended lock on the Map containing the MapItem.
//
// Contains unexported fields.
type MapItem[V any] struct {
	Value      V
	lastAccess atomic.Int64
}

// Touch will update the timestamp of a map item to the current time, extending its time to live.
// Touch is safe for concurrent use.
func (i *MapItem[V]) Touch() {
	i.lastAccess.Store(time.Now().UnixNano())
}

// Map is a "time-to-live" map such that after a given amount of time, items in the map are deleted.
//
// When a Load() or Store() occurs, the lastAccess time is set to time.Now().UnixNano(),
// Therefore, only items that are not called by Load() or Store() will be deleted after the TTL
// occurs.
//
// LoadPassive() can be used in which case the lastAccess time will *not* be updated.
//
// Map is generally safe for concurrent use, except where noted in a Map method.
//
// Adapted from: https://stackoverflow.com/a/25487392/452281
//
// Contains unexported fields.
type Map[K comparable, V any] struct {
	m       map[K]*MapItem[V]
	mtx     sync.RWMutex
	refresh bool
	stop    chan bool
}

// NewMap returns a new Map[K,V] with items expiring according to the maxTTL specified if
// they have not been accessed within that duration. Access refresh can be overridden so that
// items expire after the TTL whether they have been accessed or not.
func NewMap[K comparable, V any](
	maxTTL time.Duration,
	length int,
	pruneInterval time.Duration,
	refreshLastAccessOnGet bool,
) (m *Map[K, V]) {
	if length < 0 {
		length = 0
	}

	m = &Map[K, V]{
		m:       make(map[K]*MapItem[V], length),
		refresh: refreshLastAccessOnGet,
		stop:    make(chan bool),
	}

	ticker := time.NewTicker(pruneInterval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-m.stop:
				return
			case now := <-ticker.C:
				currentTime := now.UnixNano()
				m.mtx.Lock()
				for k, item := range m.m {
					if currentTime-item.lastAccess.Load() < int64(maxTTL) {
						continue
					}

					delete(m.m, k)
				}
				m.mtx.Unlock()
			}
		}
	}()

	return
}

// Length returns the current length of the Map's internal map. Length is safe for concurrent use.
func (m *Map[K, V]) Length() int {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return len(m.m)
}

// Store will insert a value into the Map. Store is safe for concurrent use.
func (m *Map[K, V]) Store(key K, value V) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	it, ok := m.m[key]
	if !ok {
		it = &MapItem[V]{Value: value}
		m.m[key] = it
	}

	it.Touch()
}

// Load will retrieve a value from the Map, as well as a bool indicating whether the key was
// found. If the item was not found the value returned is undefined. Load is safe for concurrent
// use.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	return m.loadImpl(key, true)
}

// LoadPassive will retrieve a value from the Map without updating that value's time to live, as
// well as a bool indicating whether the key was found. If the item was not found the value
// returned is undefined. LoadPassive is safe for concurrent use.
func (m *Map[K, V]) LoadPassive(key K) (value V, ok bool) {
	return m.loadImpl(key, false)
}

func (m *Map[K, V]) loadImpl(key K, update bool) (value V, ok bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	var it *MapItem[V]

	if it, ok = m.m[key]; !ok {
		return
	}

	value = it.Value

	if !update || !m.refresh {
		return
	}

	it.Touch()

	return
}

// Delete will remove a key and its value from the Map. Delete is safe for concurrent use.
func (m *Map[K, V]) Delete(key K) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	delete(m.m, key)
}

// Clear will remove all key/value pairs from the Map. Clear is safe for concurrent use.
func (m *Map[K, V]) Clear() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	clear(m.m)
}

// All will acquire a read lock on the internal map and returns a pointer to it, along with a
// function to use to release the read lock. Failing to release the read lock will block future
// writes and would result in a panic if the same goroutine attempts to acquire a second read lock,
// so be careful. It is also advisable to assign nil to the returned pointer after unlocking to
// prevent accidental access of map while unlocked.
//
// Generally if you only require access to the internal map to range over its values, consider
// using the Range method instead.
func (m *Map[K, V]) All() (*map[K]*MapItem[V], MapRUnlockFunc) {
	m.mtx.RLock()
	var f MapRUnlockFunc = func() { m.mtx.RUnlock() }

	return &m.m, f
}

// Eject will return a new map[K]V, mirroring the contents of the original Map. If V is a scalar
// type (integers, floats, bool) or string, Eject is safe for concurrent use. However, if the
// V is a pointer or mutable reference type (slices, maps), or a struct containing pointer or
// mutable reference types, then the values should not be considered safe for concurrent use in
// most cases.
//
// Generally if V is not a scalar or string, use the All method instead. If you only require
// access to the internal map to range over its values, consider using the Range method instead.
func (m *Map[K, V]) Eject() map[K]V {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	dst := make(map[K]V, len(m.m))

	for key, item := range m.m {
		dst[key] = item.Value
	}

	return dst
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range
// stops the iteration.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for k, item := range m.m {
		if !f(k, item.Value) {
			break
		}
	}
}

// Close will terminate TTL pruning of the Map. If Close is not called on a Map after its no longer
// needed, the Map will leak.
func (m *Map[K, V]) Close() {
	close(m.stop)
}
