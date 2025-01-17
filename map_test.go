package ttl_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/suite"

	"github.com/glenvan/ttl/v2"
)

type MapTestSuite struct {
	suite.Suite

	maxTTL        time.Duration
	pruneInterval time.Duration
	startSize     int
	sleepTime     time.Duration
	leakTestFunc  func()
}

func (s *MapTestSuite) SetupSuite() {
	s.maxTTL = 300 * time.Millisecond
	s.pruneInterval = 100 * time.Millisecond
	s.startSize = 3
	s.sleepTime = s.maxTTL + 2*s.pruneInterval
}

func (s *MapTestSuite) SetupTest() {
	s.leakTestFunc = leaktest.Check(s.T())
}

func (s *MapTestSuite) TearDownTest() {
	s.leakTestFunc()
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}

func (s *MapTestSuite) TestAllItemsExpired() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.sleepTime)

	s.Zero(tm.Length())
}

func (s *MapTestSuite) TestClose() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	tm.Close()

	time.Sleep(s.sleepTime)

	s.Equal(2, tm.Length())
}

func (s *MapTestSuite) TestCancelContext() {
	refreshOnLoad := true
	cancellableCtx, cancelFunc := context.WithCancel(context.Background())
	tm := ttl.NewMapContext[string, any](
		cancellableCtx,
		s.maxTTL,
		s.startSize,
		s.pruneInterval,
		refreshOnLoad)

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	cancelFunc()

	time.Sleep(s.sleepTime)

	s.Equal(2, tm.Length())
}

func (s *MapTestSuite) TestCloseAndCancelContextOrder() {
	refreshOnLoad := true
	cancellableCtx, cancelFunc := context.WithCancel(context.Background())
	tm := ttl.NewMapContext[string, any](
		cancellableCtx,
		s.maxTTL,
		s.startSize,
		s.pruneInterval,
		refreshOnLoad)

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.sleepTime)

	s.Equal(0, tm.Length())

	tm.Close()
	cancelFunc()
}

func (s *MapTestSuite) TestCancelContextAndCloseOrder() {
	refreshOnLoad := true
	cancellableCtx, cancelFunc := context.WithCancel(context.Background())
	tm := ttl.NewMapContext[string, any](
		cancellableCtx,
		s.maxTTL,
		s.startSize,
		s.pruneInterval,
		refreshOnLoad)

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.sleepTime)

	s.Equal(0, tm.Length())

	cancelFunc()
	tm.Close()
}

func (s *MapTestSuite) TestCancelContextAndCloseMany() {
	refreshOnLoad := true
	cancellableCtx, cancelFunc := context.WithCancel(context.Background())
	tm := ttl.NewMapContext[string, any](
		cancellableCtx,
		s.maxTTL,
		s.startSize,
		s.pruneInterval,
		refreshOnLoad)

	for i := 0; i < 10; i++ {
		cancelFunc()
		tm.Close()
	}

	cancelFunc() // one more to satisfy the golang static analysis
}

func (s *MapTestSuite) TestNoItemsExpired() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	time.Sleep(s.maxTTL - 100*time.Millisecond)
	s.Equal(2, tm.Length())
}

func (s *MapTestSuite) TestTTLRefresh() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	refreshedKey := "int"

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})
	tm.Store(refreshedKey, 1234)

	doneCh := make(chan struct{})

	go func() {
		for start := time.Now(); time.Since(start) < s.sleepTime; {
			time.Sleep(50 * time.Millisecond)
			tm.Load(refreshedKey)
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
	refreshOnLoad := false
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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

func (s *MapTestSuite) TestStoreExistingWritesNewValue() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store(0, 0)
	tm.Store(0, 1)
	tm.Store(0, 2)

	tm.Store(1, 1)
	tm.Store(1, 2)

	s.Equal(2, tm.Length())
	v, ok := tm.Load(0)
	if s.True(ok) {
		s.Equal(2, v)
	}
	v, ok = tm.Load(1)
	if s.True(ok) {
		s.Equal(2, v)
	}
}

func (s *MapTestSuite) TestStoreWithTTLExistingWritesNewValue() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.StoreWithTTL(0, 0, time.Minute)
	tm.StoreWithTTL(0, 1, time.Minute)
	tm.StoreWithTTL(0, 2, time.Minute)

	tm.StoreWithTTL(1, 1, time.Minute)
	tm.StoreWithTTL(1, 2, time.Minute)

	s.Equal(2, tm.Length())
	v, ok := tm.Load(0)
	if s.True(ok) {
		s.Equal(2, v)
	}
	v, ok = tm.Load(1)
	if s.True(ok) {
		s.Equal(2, v)
	}
}

func (s *MapTestSuite) TestStoreAndStoreWithTTLExistingWritesNewValue() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store(0, 0)
	tm.Store(0, 1)
	tm.StoreWithTTL(0, 2, time.Minute)

	tm.StoreWithTTL(1, 1, time.Minute)
	tm.Store(1, 2)

	s.Equal(2, tm.Length())
	v, ok := tm.Load(0)
	if s.True(ok) {
		s.Equal(2, v)
	}
	v, ok = tm.Load(1)
	if s.True(ok) {
		s.Equal(2, v)
	}
}

func (s *MapTestSuite) TestStoreWithTTL() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.StoreWithTTL("int slice", []int{1, 2, 3}, 2*s.maxTTL)

	time.Sleep(s.sleepTime)

	s.Equal(1, tm.Length())
	_, ok := tm.LoadPassive("int slice")
	s.True(ok)

	time.Sleep(s.sleepTime)

	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestStoreDoesNotResetCustomTTL() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.StoreWithTTL("int slice", []int{1, 2, 3}, 2*s.maxTTL)
	updatedSlice := []int{3, 2, 1}
	tm.Store("int slice", updatedSlice)

	time.Sleep(s.sleepTime)

	s.Equal(1, tm.Length())
	v, ok := tm.LoadPassive("int slice")
	if s.True(ok) {
		value, ok := v.([]int)
		if s.True(ok) {
			s.Truef(
				slices.Equal(updatedSlice, value),
				"Actual %v does not equal expected %v",
				value,
				updatedSlice)
		}
	}

	time.Sleep(s.sleepTime)

	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestDelete() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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

func (s *MapTestSuite) TestDeleteFuncByKey() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, string](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store(0, "zero")
	tm.Store(1, "one")
	tm.Store(2, "two")
	tm.Store(3, "three")

	s.Equal(4, tm.Length())

	// Delete all even keys
	tm.DeleteFunc(func(key int, val string) bool {
		return key%2 == 0
	})

	if s.Equal(2, tm.Length()) {
		v, ok := tm.Load(1)
		if s.True(ok) {
			s.Equal("one", v)
		}

		v, ok = tm.Load(3)
		if s.True(ok) {
			s.Equal("three", v)
		}
	}
}

func (s *MapTestSuite) TestDeleteFuncByValue() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("zero", 0)
	tm.Store("one", 1)
	tm.Store("two", 2)
	tm.Store("three", 3)

	s.Equal(4, tm.Length())

	// Delete all even keys
	tm.DeleteFunc(func(key string, val int) bool {
		return val%2 == 0
	})

	s.Equal(2, tm.Length())

	if s.Equal(2, tm.Length()) {
		v, ok := tm.Load("one")
		if s.True(ok) {
			s.Equal(1, v)
		}

		v, ok = tm.Load("three")
		if s.True(ok) {
			s.Equal(3, v)
		}
	}
}

func (s *MapTestSuite) TestClear() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store("myString", "a b c")
	tm.Store("int slice", []int{1, 2, 3})

	s.Equal(2, tm.Length())

	tm.Clear()

	s.Equal(0, tm.Length())
}

func (s *MapTestSuite) TestLoadPassive() {
	refreshOnLoad := true
	tm := ttl.NewMap[string, any](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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
	refreshOnLoad := true
	tm := ttl.NewMap[uint64, string](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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
	refreshOnLoad := true
	tm := ttl.NewMap[float32, string](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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
	refreshOnLoad := true
	tm := ttl.NewMap[byte, string](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	tm.Store(0x41, "A")
	tm.Store(0x7a, "z")

	s.Equal(2, tm.Length())

	v, ok := tm.Load(0x41)
	if s.True(ok) {
		s.Equal("A", v)
	}
}

func (s *MapTestSuite) TestStructKey() {
	type pos struct {
		x int
		y int
	}

	refreshOnLoad := true
	tm := ttl.NewMap[pos, string](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	topLeft := pos{0, 0}
	bottomRight := pos{100, 200}

	tm.Store(topLeft, "top left")
	tm.Store(bottomRight, "bottom right")

	s.Equal(2, tm.Length())

	v, ok := tm.Load(bottomRight)
	if s.True(ok) {
		s.Equal("bottom right", v)
	}
}

func (s *MapTestSuite) TestRange() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, []int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
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

func (s *MapTestSuite) TestRangeDelete() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, []int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	firstSlice := []int{1, 2, 3}
	secondSlice := []int{4, 5, 6}
	thirdSlice := []int{7, 8, 9}
	tm.Store(0, firstSlice)
	tm.Store(1, secondSlice)
	tm.Store(2, thirdSlice)

	s.Equal(3, tm.Length())

	var wg sync.WaitGroup

	// Delete all even keys
	tm.Range(func(key int, val []int) bool {
		if key%2 == 0 {
			// deferred deletion in a goroutine
			wg.Add(1)
			go func() {
				tm.Delete(key)
				wg.Done()
			}()
		}

		return true // continue
	})

	wg.Wait()

	s.Equal(1, tm.Length())
}

func (s *MapTestSuite) TestRangeTransform() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, []int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	firstSlice := []int{1, 2, 3}
	secondSlice := []int{4, 5, 6}
	thirdSlice := []int{7, 8, 9}
	tm.Store(0, firstSlice)
	tm.Store(1, secondSlice)
	tm.Store(2, thirdSlice)

	s.Equal(3, tm.Length())

	// Transform in-place
	tm.Range(func(key int, val []int) bool {
		for k, v := range val {
			val[k] = v * 2
		}

		return true
	})

	s.Equal(3, tm.Length())

	transformedThirdSlice := []int{14, 16, 18}

	v, ok := tm.Load(2)
	if s.True(ok) {
		s.Truef(
			slices.Equal(transformedThirdSlice, v),
			"Actual %v does not equal expected %v",
			v,
			transformedThirdSlice)
	}
}

func (s *MapTestSuite) TestConcurrentStoreExpire() {
	refreshOnLoad := true
	tm := ttl.NewMap[uint64, uint64](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	defer tm.Close()

	startTime := time.Now()

	doneCh := make(chan struct{})

	go func() {
		var iteration uint64
		for time.Since(startTime) < s.maxTTL {
			tm.Store(iteration, iteration)
			iteration++
		}

		time.Sleep(s.sleepTime)
		close(doneCh)
	}()

	<-doneCh

	s.Zero(tm.Length())
}

func (s *MapTestSuite) TestConcurrentStoreDelete() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	tm.Close() // disable pruning

	startTime := time.Now()

	storedCh := make(chan int, 1000)
	var iteration int

	go func() {
		for time.Since(startTime) < s.maxTTL {
			tm.Store(iteration, iteration)
			storedCh <- iteration
			iteration++
		}

		close(storedCh)
	}()

	for storedKey := range storedCh {
		tm.Delete(storedKey)
	}

	s.Zero(tm.Length())
	s.Greater(iteration, 0)
}

func (s *MapTestSuite) TestConcurrentStoreDeleteFuncOne() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	tm.Close() // disable pruning

	startTime := time.Now()

	storedCh := make(chan int, 1000)
	var iteration int

	go func() {
		for time.Since(startTime) < s.maxTTL {
			tm.Store(iteration, iteration)
			storedCh <- iteration
			iteration++
		}

		close(storedCh)
	}()

	for storedKey := range storedCh {
		tm.DeleteFunc(func(key int, _ int) bool {
			return key == storedKey
		})
	}

	s.Zero(tm.Length())
	s.Greater(iteration, 0)
}

func (s *MapTestSuite) TestConcurrentStoreDeleteFuncMatch() {
	refreshOnLoad := true
	tm := ttl.NewMap[int, int](s.maxTTL, s.startSize, s.pruneInterval, refreshOnLoad)
	tm.Close() // disable pruning

	startTime := time.Now()

	storedCh := make(chan int, 1000)
	iteration := 0

	go func() {
		for time.Since(startTime) < s.maxTTL {
			tm.Store(iteration+1, iteration+1)
			storedCh <- iteration + 1
			iteration++
		}

		close(storedCh)
	}()

	for range storedCh {
		tm.DeleteFunc(func(key int, _ int) bool {
			// Delete all odd keys
			return key%2 == 1
		})
	}

	if s.Greater(iteration, 0) {
		// Since we're deleting odd keys this should never be off by one if iteration isn't
		// an even number
		s.Equal(iteration/2, tm.Length())
	}
}
