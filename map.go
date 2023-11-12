package ttl

import (
	"sync"
	"sync/atomic"
	"time"
)

type mapItem[V any] struct {
	Value      V
	lastAccess atomic.Int64
}

func (i *mapItem[V]) touch() {
	i.lastAccess.Store(time.Now().UnixNano())
}

// Map is a "time-to-live" map such that after a given amount of time, items in the map are deleted.
// Map is safe for concurrent use.
//
// When a [Map.Load] or [Map.Store] occurs, the lastAccess time is set to the current time.
// Therefore, only items that are not called by [Map.Load] or [Map.Store] will be deleted after
// the TTL expires.
//
// [Map.LoadPassive] can be used in which case the lastAccess time will *not* be updated.
//
// Adapted from: https://stackoverflow.com/a/25487392/452281
type Map[K comparable, V any] struct {
	m       map[K]*mapItem[V]
	mtx     sync.RWMutex
	refresh bool
	stop    chan bool
}

// NewMap returns a new [Map] with items expiring according to the maxTTL specified if
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
		m:       make(map[K]*mapItem[V], length),
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

// Length returns the current length of the [Map]'s internal map. Length is safe for concurrent use.
func (m *Map[K, V]) Length() int {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	return len(m.m)
}

// Store will insert a value into the [Map]. Store is safe for concurrent use.
func (m *Map[K, V]) Store(key K, value V) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	it, ok := m.m[key]
	if !ok {
		it = &mapItem[V]{Value: value}
		m.m[key] = it
	}

	it.touch()
}

// Load will retrieve a value from the [Map], as well as a bool indicating whether the key was
// found. If the item was not found the value returned is undefined. Load is safe for concurrent
// use.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	return m.loadImpl(key, true)
}

// LoadPassive will retrieve a value from the [Map] (without updating that value's time to live),
// as well as a bool indicating whether the key was found. If the item was not found the value
// returned is undefined. LoadPassive is safe for concurrent use.
func (m *Map[K, V]) LoadPassive(key K) (value V, ok bool) {
	return m.loadImpl(key, false)
}

func (m *Map[K, V]) loadImpl(key K, update bool) (value V, ok bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	var it *mapItem[V]

	if it, ok = m.m[key]; !ok {
		return
	}

	value = it.Value

	if !update || !m.refresh {
		return
	}

	it.touch()

	return
}

// Delete will remove a key and its value from the [Map]. Delete is safe for concurrent use.
func (m *Map[K, V]) Delete(key K) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	delete(m.m, key)
}

// Clear will remove all key/value pairs from the [Map]. Clear is safe for concurrent use.
func (m *Map[K, V]) Clear() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	clear(m.m)
}

// Range calls f sequentially for each key and value present in the [Map]. If f returns false, range
// stops the iteration.
//
// Range is safe for concurrent use and supports modifying the value (assuming it's a slice, map,
// or a struct) within the range function. However, this requires a write lock on the Map – so you
// are not able to perform [Map.Delete] or [Map.Store] operations on the original [Map] directly
// within the range func, as that would cause a panic. Even an accessor like [Map.Load] or
// [Map.LoadPassive] would lock indefinitely.
//
// If you need to perform operations on the original [Map], do so in a new goroutine from within
// the range func – effectively deferring the operation until the Range completes.
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
