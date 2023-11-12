package ttl_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/glenvan/ttl"
)

type MapTestSuite struct {
	suite.Suite

	maxTTL        time.Duration
	pruneInterval time.Duration
	startSize     int
	sleepTime     time.Duration
}

func (s *MapTestSuite) SetupSuite() {
	s.maxTTL = 300 * time.Millisecond
	s.pruneInterval = 100 * time.Millisecond
	s.startSize = 3
	s.sleepTime = s.maxTTL + 2*s.pruneInterval
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}

func (s *MapTestSuite) TestAllItemsExpired() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.maxTTL + s.pruneInterval)

	s.Zero(tm.Length())
}

func (s *MapTestSuite) TestNoItemsExpired() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.maxTTL - 100*time.Millisecond)
	s.Equal(2, tm.Length())
}

func (s *MapTestSuite) TestTTLRefresh() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	dontExpireKey := "int"

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})
	tm.Store(dontExpireKey, 1234)

	doneCh := make(chan struct{})

	go func() {
		for start := time.Now(); time.Since(start) < s.sleepTime; {
			time.Sleep(50 * time.Millisecond)
			tm.Load(dontExpireKey)
		}
		close(doneCh)
	}()

	<-doneCh

	s.Equal(1, tm.Length())

	tm.Range(func(key string, value any) bool {
		v, ok := value.(int)
		if s.True(ok) {
			s.Equal(1234, v)
		}

		return true
	})
}

func (s *MapTestSuite) TestWithNoRefresh() {
	refreshLastAccessOnGet := false
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	doneCh := make(chan struct{})

	go func() {
		for start := time.Now(); time.Since(start) < s.sleepTime; {
			time.Sleep(50 * time.Millisecond)

			tm.Load("myString")
			tm.Load("int slice")
		}
		close(doneCh)
	}()

	<-doneCh

	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestDelete() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	s.Equal(2, tm.Length())

	tm.Delete("int slice")

	_, ok := tm.Load("nt slice")
	s.False(ok)
	s.Equal(1, tm.Length())

	tm.Delete("myString")

	_, ok = tm.Load("myString")
	s.False(ok)
	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestClear() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	s.Equal(2, tm.Length())

	tm.Clear()

	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestAllFunc() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int", 1234)
	tm.Store("float Pi", 3.1415)
	tm.Store("int slice", []int{1, 2, 3})
	tm.Store("boolean", true)

	s.Equal(5, tm.Length())

	tm.Delete("float Pi")
	_, ok := tm.Load("float Pi")
	s.False(ok)

	s.Equal(4, tm.Length())

	tm.Store("byte", "0x7b")
	var u uint64 = 123456789
	tm.Store("uint64", u)

	s.Equal(6, tm.Length())
}

func (s *MapTestSuite) TestLoadPassive() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int", 1234)
	tm.Store("float Pi", 3.1415)
	tm.Store("int slice", []int{1, 2, 3})
	tm.Store("boolean", true)

	doneCh := make(chan struct{})

	go func() {
		for start := time.Now(); time.Since(start) < s.sleepTime; {
			time.Sleep(50 * time.Millisecond)

			tm.LoadPassive("myString")
			tm.LoadPassive("int slice")
		}
		close(doneCh)
	}()

	<-doneCh

	s.Zero(tm.Length())
}

func (s *MapTestSuite) TestUInt64Key() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[uint64, string](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(18446744073709551615, "largest")
	tm.Store(9223372036854776000, "mid")
	tm.Store(0, "zero")

	s.Equal(3, tm.Length())

	v, ok := tm.Load(9223372036854776000)
	if s.True(ok) {
		s.Equal("mid", v)
	}
}

func (s *MapTestSuite) TestUFloat32Key() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[float32, string](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(34000000000.12345, "largest")
	tm.Store(12312312312.98765, "mid")
	tm.Store(0.001, "tiny")

	s.Equal(3, tm.Length())

	v, ok := tm.Load(0.001)
	if s.True(ok) {
		s.Equal("tiny", v)
	}
}

func (s *MapTestSuite) TestByteKey() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[byte, string](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	tm.Store(0x41, "A")
	tm.Store(0x7a, "z")

	s.Equal(2, tm.Length())

	v, ok := tm.Load(0x41)
	if s.True(ok) {
		s.Equal("A", v)
	}
}

func (s *MapTestSuite) TestRange() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[int, []int](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	firstSlice := []int{1, 2, 3}
	secondSlice := []int{4, 5, 6}
	tm.Store(0, firstSlice)
	tm.Store(1, secondSlice)

	s.Equal(2, tm.Length())

	tm.Range(func(key int, val []int) bool {
		switch key {
		case 0:
			s.True(slices.Equal(firstSlice, val), "firstSlice did not equal the returned value")

		case 1:
			s.True(slices.Equal(secondSlice, val), "secondSlice did not equal the returned value")
		}

		return true
	})
}

func (s *MapTestSuite) TestRangeMutate() {
	refreshLastAccessOnGet := true
	tm := ttl.NewMap[int, []int](s.maxTTL, s.startSize, s.pruneInterval, refreshLastAccessOnGet)
	defer tm.Close()

	firstSlice := []int{1, 2, 3}
	secondSlice := []int{4, 5, 6}
	thirdSlice := []int{7, 8, 9}
	tm.Store(0, firstSlice)
	tm.Store(1, secondSlice)
	tm.Store(2, thirdSlice)

	s.Equal(3, tm.Length())

	doneCh := make(chan struct{})

	tm.Range(func(key int, val []int) bool {
		switch key {
		case 0:
			// deferred deletion in a goroutine
			go func() {
				tm.Delete(key)
				close(doneCh)
			}()

		case 2:
			val[1] = 88
		}

		return true
	})

	<-doneCh

	modifiedThirdSlice := []int{7, 88, 9}
	s.Equal(2, tm.Length())
	v, ok := tm.Load(2)
	if s.True(ok) {
		s.Truef(
			slices.Equal(modifiedThirdSlice, v),
			"Actual %v does not equal expected %v",
			v,
			modifiedThirdSlice)
	}
}
