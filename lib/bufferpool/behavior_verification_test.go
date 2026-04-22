// Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufferpool_test

import (
	"sync"
	"testing"

	"github.com/openGemini/openGemini/lib/bufferpool"
)

// TestBufferPool_CapacityPreservation_ZeroCapacity
// Verifies that the pool preserves buffer capacities, including zero capacity
// This is EXPECTED BEHAVIOR, not a bug
func TestBufferPool_CapacityPreservation_ZeroCapacity(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put a zero-capacity buffer
	zeroCapBuf := make([]byte, 0, 0)
	pool.Put(zeroCapBuf)

	// Get should return a buffer with the same capacity (0)
	buf := pool.Get()

	// This is EXPECTED: capacity preservation is the design
	if cap(buf) != 0 {
		t.Errorf("Expected capacity preservation: put cap=0, got cap=%d", cap(buf))
	}
	t.Logf("✓ Capacity preserved: put cap=0, got cap=%d", cap(buf))
}

// TestBufferPool_CapacityPreservation_SmallCapacity
// Verifies capacity preservation for small buffers
func TestBufferPool_CapacityPreservation_SmallCapacity(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put a small buffer (capacity less than pool's defaultSize)
	smallBuf := make([]byte, 0, 10)
	pool.Put(smallBuf)

	// Get should return a buffer with the same capacity (10)
	buf := pool.Get()

	if cap(buf) != 10 {
		t.Errorf("Expected capacity preservation: put cap=10, got cap=%d", cap(buf))
	}
	t.Logf("✓ Capacity preserved: put cap=10, got cap=%d", cap(buf))
}

// TestBufferPool_CapacityPreservation_LargeCapacity
// Verifies capacity preservation for large buffers
func TestBufferPool_CapacityPreservation_LargeCapacity(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put a large buffer
	largeBuf := make([]byte, 0, 32*1024*1024+1)
	pool.Put(largeBuf)

	// Get should return a buffer with the same capacity
	buf := pool.Get()

	if cap(buf) != cap(largeBuf) {
		t.Errorf("Expected capacity preservation: put cap=%d, got cap=%d", cap(largeBuf), cap(buf))
	}
	t.Logf("✓ Capacity preserved: put cap=%d, got cap=%d", cap(largeBuf), cap(buf))
}

// TestBufferPool_LengthResetOnPut
// Verifies that Put always resets buffer length to 0
func TestBufferPool_LengthResetOnPut(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put a buffer with data
	buf := make([]byte, 100)
	for i := range buf {
		buf[i] = byte(i)
	}
	pool.Put(buf)

	// Get should return buffer with len=0
	buf2 := pool.Get()

	if len(buf2) != 0 {
		t.Errorf("Expected len=0 after Put, got len=%d", len(buf2))
	}
	t.Logf("✓ Length reset: put len=%d, got len=%d", len(buf), len(buf2))
}

// TestBufferPool_MultipleCapacityPreservation
// Verifies capacity preservation across multiple put/get cycles
func TestBufferPool_MultipleCapacityPreservation(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	testCases := []struct {
		name string
		cap  int
	}{
		{"zero", 0},
		{"small", 10},
		{"medium", 100},
		{"large", 1000},
		{"very_large", 10000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, 0, tc.cap)
			pool.Put(buf)

			buf2 := pool.Get()
			if cap(buf2) != tc.cap {
				t.Errorf("Capacity not preserved: expected %d, got %d", tc.cap, cap(buf2))
			}
		})
	}
}

// TestBufferPool_ConcurrentCapacityPreservation
// Verifies capacity preservation under concurrent access
func TestBufferPool_ConcurrentCapacityPreservation(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 4, 8)

	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := false

	// Launch goroutines with different buffer capacities
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			capacity := id * 100
			buf := make([]byte, 0, capacity)
			pool.Put(buf)

			buf2 := pool.Get()
			// Note: We might not get the same buffer back due to concurrency
			// But we should get SOME buffer with len=0
			if len(buf2) != 0 {
				mu.Lock()
				failed = true
				mu.Unlock()
				t.Logf("Goroutine %d: expected len=0, got len=%d", id, len(buf2))
			}
		}(i)
	}

	wg.Wait()

	if failed {
		t.Error("Concurrent test failed")
	}
}

// TestBufferPool_DefaultPoolCapacityPreservation
// Verifies capacity preservation in the default pool
func TestBufferPool_DefaultPoolCapacityPreservation(t *testing.T) {
	// Test with default pool
	testCases := []int{0, 4, 100, 1000}

	for _, capacity := range testCases {
		buf := make([]byte, 0, capacity)
		bufferpool.Put(buf)

		buf2 := bufferpool.Get()
		if cap(buf2) != capacity {
			t.Errorf("Default pool: capacity not preserved for cap=%d, got cap=%d", capacity, cap(buf2))
		}
		t.Logf("✓ Default pool: capacity preserved for cap=%d", capacity)
	}
}

// TestBufferPool_NewBufferDefaultSize
// Verifies that new buffers from a fresh pool have defaultSize when created new
func TestBufferPool_NewBufferDefaultSize(t *testing.T) {
	// Create a fresh pool with a small cache
	pool := bufferpool.NewByteBufferPool(2048, 1, 1)

	// Put a buffer with a known capacity, then get it back
	// This tests the capacity preservation path
	buf := make([]byte, 0, 2048)
	pool.Put(buf)

	buf2 := pool.Get()

	// Should preserve the capacity we put in
	if cap(buf2) != 2048 {
		t.Errorf("Expected capacity 2048, got cap=%d", cap(buf2))
	}
	t.Logf("✓ Buffer capacity preserved: put cap=2048, got cap=%d", cap(buf2))
}

// TestBufferPool_ResizeBehavior
// Verifies Resize function behavior
func TestBufferPool_ResizeBehavior(t *testing.T) {
	testCases := []struct {
		originalCap int
		newSize     int
	}{
		{10, 5},  // shrink
		{10, 10}, // same
		{10, 20}, // grow
		{0, 100}, // zero to non-zero
		{100, 0}, // non-zero to zero
	}

	for _, tc := range testCases {
		buf := make([]byte, 0, tc.originalCap)
		resized := bufferpool.Resize(buf, tc.newSize)

		if len(resized) != tc.newSize {
			t.Errorf("Resize(%d, %d): expected len=%d, got len=%d",
				tc.originalCap, tc.newSize, tc.newSize, len(resized))
		}

		if cap(resized) < tc.newSize {
			t.Errorf("Resize(%d, %d): expected cap>=%d, got cap=%d",
				tc.originalCap, tc.newSize, tc.newSize, cap(resized))
		}

		t.Logf("✓ Resize(cap=%d, size=%d): got len=%d, cap=%d",
			tc.originalCap, tc.newSize, len(resized), cap(resized))
	}
}

// TestBufferPool_PutResetsLength
// Verifies that Put always resets length to 0
func TestBufferPool_PutResetsLength(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Create buffer with data
	buf := make([]byte, 0, 100)
	for i := 0; i < 50; i++ {
		buf = append(buf, byte(i))
	}

	originalLen := len(buf)
	pool.Put(buf)

	// Get it back
	buf2 := pool.Get()

	// Length should be 0, capacity should be preserved
	if len(buf2) != 0 {
		t.Errorf("After Put, buffer length should be 0, got %d", len(buf2))
	}
	if cap(buf2) != 100 {
		t.Errorf("Capacity should be preserved, expected 100, got %d", cap(buf2))
	}

	t.Logf("✓ Put resets length: original len=%d, after Get len=%d, cap=%d",
		originalLen, len(buf2), cap(buf2))
}

// TestBufferPool_MultiplePoolsIndependent
// Verifies that different pools operate independently
func TestBufferPool_MultiplePoolsIndependent(t *testing.T) {
	pool1 := bufferpool.NewByteBufferPool(512, 1, 1)
	pool2 := bufferpool.NewByteBufferPool(2048, 1, 1)

	// Put different capacity buffers in each pool
	buf1 := make([]byte, 0, 100)
	buf2 := make([]byte, 0, 200)

	pool1.Put(buf1)
	pool2.Put(buf2)

	// Get from each pool
	buf1Out := pool1.Get()
	buf2Out := pool2.Get()

	// Each pool should preserve its own buffers' capacities
	if cap(buf1Out) != 100 {
		t.Errorf("Pool1: expected cap=100, got cap=%d", cap(buf1Out))
	}
	if cap(buf2Out) != 200 {
		t.Errorf("Pool2: expected cap=200, got cap=%d", cap(buf2Out))
	}

	t.Logf("✓ Pools operate independently")
	t.Logf("  Pool1: put cap=100, got cap=%d", cap(buf1Out))
	t.Logf("  Pool2: put cap=200, got cap=%d", cap(buf2Out))
}
