package ttl_test

import (
	"testing"
	"time"

	"github.com/glenvan/ttl"
)

func TestAllItemsExpired(t *testing.T) {
	maxTTL := time.Duration(time.Second * 4)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1 2 3")

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() > 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}

func TestNoItemsExpired(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 3) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1 2 3")

	time.Sleep(maxTTL)
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 2 {
		t.Fatalf("t.Length should equal 2, but actually equals %v\n", tm.Length())
	}
}

func TestKeepFloat(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	dontExpireKey := "int"

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1 2 3")
	tm.Store(dontExpireKey, "1234")

	go func() {
		for range time.Tick(time.Second) {
			tm.Load(dontExpireKey)
		}
	}()

	time.Sleep(maxTTL + pruneInterval)
	if tm.Length() != 1 {
		t.Fatalf("t.Length should equal 1, but actually equals %v\n", tm.Length())
	}

	allItems, unlockFunc := tm.All()
	defer unlockFunc()
	if (*allItems)[dontExpireKey].Value != "1234" {
		t.Errorf("Value should equal 1234 but actually equals %v\n", (*allItems)[dontExpireKey].Value)
	}
	t.Logf("tm.Length: %v\n", tm.Length())
	t.Logf("%v Value: %v\n", dontExpireKey, (*allItems)[dontExpireKey].Value)

}

func TestWithNoRefresh(t *testing.T) {
	maxTTL := time.Duration(time.Second * 4)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 1) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := false                 // do NOT update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1, 2, 3")

	go func() {
		for range time.Tick(time.Second) {
			tm.Load("myString")
			tm.Load("int_array")
		}
	}()

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.Length: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}

func TestDelete(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1, 2, 3")

	tm.Delete("int_array")
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 1 {
		t.Fatalf("t.Length should equal 1, but actually equals %v\n", tm.Length())
	}

	tm.Delete("myString")
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Fatalf("t.Length should equal 0, but actually equals %v\n", tm.Length())
	}
}

func TestClear(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int_array", "1, 2, 3")
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 2 {
		t.Fatalf("t.Length should equal 2, but actually equals %v\n", tm.Length())
	}

	tm.Clear()
	t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Fatalf("t.Length should equal 0, but actually equals %v\n", tm.Length())
	}
}

func TestAllFunc(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int", "1234")
	tm.Store("floatPi", "3.1415")
	tm.Store("int_array", "1, 2, 3")
	tm.Store("boolean", "true")

	tm.Delete("floatPi")
	//t.Logf("tm.len: %v\n", tm.Length())
	if tm.Length() != 4 {
		t.Fatalf("t.Length should equal 4, but actually equals %v\n", tm.Length())
	}

	tm.Store("byte", "0x7b")
	var u = "123456789"
	tm.Store("uint64", u)

	allItems, unlockFunc := tm.All()
	defer unlockFunc()
	if (*allItems)["int"].Value != "1234" {
		t.Fatalf("allItems and tm.m are not equal\n")
	}
}

func TestGetNoUpdate(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[string, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	// populate the TtlMap
	tm.Store("myString", "a b c")
	tm.Store("int", "1234")
	tm.Store("floatPi", "3.1415")
	tm.Store("int_array", "1, 2, 3")
	tm.Store("boolean", "true")

	go func() {
		for range time.Tick(time.Second) {
			tm.LoadPassive("myString")
			tm.LoadPassive("int_array")
		}
	}()

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.Length: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}

func TestUInt64Key(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[uint64, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(18446744073709551615, "largest")
	tm.Store(9223372036854776000, "mid")
	tm.Store(0, "zero")

	allItems, unlockFunc := tm.All()
	for k, v := range *allItems {
		t.Logf("k: %v   v: %v\n", k, v.Value)
	}
	unlockFunc()
	allItems = nil

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.Length: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}

func TestUFloat32Key(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[float32, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(34000000000.12345, "largest")
	tm.Store(12312312312.98765, "mid")
	tm.Store(0.001, "tiny")

	allItems, unlockFunc := tm.All()
	for k, v := range *allItems {
		t.Logf("k: %v   v: %v\n", k, v.Value)
	}
	unlockFunc()
	allItems = nil

	v, ok := tm.Load(0.001)
	t.Logf("k: 0.001   v:%v   (verified: %t)\n", v, ok)

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.Length: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}

func TestByteKey(t *testing.T) {
	maxTTL := time.Duration(time.Second * 2)        // time in seconds
	startSize := 3                                  // initial number of items in map
	pruneInterval := time.Duration(time.Second * 4) // search for expired items every 'pruneInterval' seconds
	refreshLastAccessOnGet := true                  // update item's lastAccessTime on a .Get()
	tm := ttl.NewMap[byte, string](maxTTL, startSize, pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(0x41, "A")
	tm.Store(0x7a, "z")

	allItems, unlockFunc := tm.All()
	for k, v := range *allItems {
		t.Logf("k: %x   v: %v\n", k, v.Value)
	}
	unlockFunc()
	allItems = nil

	time.Sleep(maxTTL + pruneInterval)
	t.Logf("tm.Length: %v\n", tm.Length())
	if tm.Length() != 0 {
		t.Errorf("t.Length should be 0, but actually equals %v\n", tm.Length())
	}
}
