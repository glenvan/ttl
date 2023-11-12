package ttl

import (
	"context"
	"maps"
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
	closed  atomic.Bool
}

// NewMap returns a new [Map] with items expiring according to the maxTTL specified if
// they have not been accessed within that duration. Access refresh can be overridden so that
// items expire after the TTL whether they have been accessed or not.
//
// NewMap accepts a context. If the context is cancelled, the pruning process will automatically
// stop whether you've called [Map.Close] or not. It's safe to use either approach.
//
// context.Background() is perfectly acceptable as the default context, however you should
// [Map.Close] the Map yourself in that case.
func NewMap[K comparable, V any](
	ctx context.Context,
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

	go func() {
		ticker := time.NewTicker(pruneInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.Close()
				return
			case <-m.stop:
				return
			case now := <-ticker.C:
				currentTime := now.UnixNano()
				m.mtx.Lock()
				maps.DeleteFunc(m.m, func(key K, item *mapItem[V]) bool {
					return currentTime-item.lastAccess.Load() >= int64(maxTTL)
				})
				m.mtx.Unlock()
			}
		}
	}()

	return
}

// Close will terminate TTL pruning of the Map. If Close is not called on a Map after it's no longer
// needed, the Map will leak (unless the context has been cancelled).
//
// Close may be called multiple times and is safe to call even if the context has been cancelled.
func (m *Map[K, V]) Close() {
	if m.closed.CompareAndSwap(false, true) {
		close(m.stop)
	}
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

// DeleteFunc deletes any key/value pairs from the [Map] for which del returns true. DeleteFunc is
// safe for concurrent use.
func (m *Map[K, V]) DeleteFunc(del func(key K, value V) bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for key, item := range m.m {
		if del(key, item.Value) {
			delete(m.m, key)
		}
	}
}

// Clear will remove all key/value pairs from the [Map]. Clear is safe for concurrent use.
func (m *Map[K, V]) Clear() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	clear(m.m)
}

// Range calls f sequentially for each key and value present in the [Map]. If f returns false, Range
// stops the iteration.
//
// Range is safe for concurrent use and supports modifying the value (assuming it's a reference
// type like a slice, map, or a pointer) within the range function. However, this requires a write
// lock on the Map – so you are not able to perform [Map.Delete] or [Map.Store] operations on the
// original [Map] directly within the range func, as that would cause a panic. Even an accessor
// like [Map.Load] or [Map.LoadPassive] would lock indefinitely.
//
// If you need to perform operations on the original [Map], do so in a new goroutine from within
// the range func – effectively deferring the operation until the Range completes.
//
// If you just need to delete items with a certain key or value, use [Map.DeleteFunc] instead.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for key, item := range m.m {
		if !f(key, item.Value) {
			break
		}
	}
}
