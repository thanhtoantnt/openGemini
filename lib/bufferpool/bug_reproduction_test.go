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

// TestBufferPool_GetReturnsZeroCapacity_BugReproduction
// This test reproduces the bug where Get() returns a buffer with zero capacity
func TestBufferPool_GetReturnsZeroCapacity_BugReproduction(t *testing.T) {
	// Create a pool with default size
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// First, put a zero-capacity buffer into the pool (this simulates the bug scenario)
	zeroCapBuf := make([]byte, 0, 0)
	pool.Put(zeroCapBuf)

	// Now try to get a buffer - this should ideally return a valid buffer
	buf := pool.Get()

	// BUG: If this returns a zero-capacity buffer, it's a bug
	if cap(buf) == 0 {
		t.Errorf("CRITICAL BUG: Get() returned buffer with zero capacity (cap=%d, len=%d)", cap(buf), len(buf))
		t.Logf("Buffer details: cap=%d, len=%d, ptr=%p", cap(buf), len(buf), buf)
	} else if cap(buf) < 64 {
		t.Errorf("Buffer capacity %d is less than expected minimum 64", cap(buf))
	} else {
		t.Logf("OK: Buffer has valid capacity %d", cap(buf))
	}
}

// TestBufferPool_GetAfterPuttingSmallBuffer_BugReproduction
// This test reproduces the bug by putting a small buffer and getting it back
func TestBufferPool_GetAfterPuttingSmallBuffer_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put a buffer with capacity less than the pool's default size
	smallBuf := make([]byte, 0, 10) // capacity 10, but pool default is 1024
	pool.Put(smallBuf)

	// Try to get a buffer - it might return the small buffer we just put
	buf := pool.Get()

	if cap(buf) < 64 {
		t.Errorf("CRITICAL BUG: Get() returned buffer with capacity %d, expected at least 64", cap(buf))
		t.Logf("Put buffer capacity: 10, Got buffer capacity: %d", cap(buf))
		t.Logf("This violates the pool's default size contract of 1024")
	}
}

// TestBufferPool_MultipleGets_BugReproduction
// This test shows the bug when multiple get/put operations occur
func TestBufferPool_MultipleGets_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Get and put several buffers with varying capacities
	for i := 0; i < 10; i++ {
		buf := pool.Get()
		// Check if we got a valid buffer
		if cap(buf) < 64 {
			t.Errorf("CRITICAL BUG at iteration %d: Get() returned buffer with capacity %d", i, cap(buf))
			return
		}
		pool.Put(buf)
	}

	// Now put a problematic buffer
	badBuf := make([]byte, 0, 0)
	pool.Put(badBuf)

	// The next get might return the bad buffer
	buf := pool.Get()
	if cap(buf) < 64 {
		t.Errorf("CRITICAL BUG: After putting zero-capacity buffer, Get() returned buffer with capacity %d", cap(buf))
	}
}

// TestBufferPool_PutThenGetRaceCondition_BugReproduction
// This test simulates a race condition scenario
func TestBufferPool_PutThenGetRaceCondition_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put several buffers
	for i := 0; i < 5; i++ {
		buf := make([]byte, 0, 32) // smaller than default
		pool.Put(buf)
	}

	// Now get buffers and check capacities
	for i := 0; i < 5; i++ {
		buf := pool.Get()
		if cap(buf) < 64 {
			t.Errorf("CRITICAL BUG at get %d: Got buffer with capacity %d, expected >= 64", i, cap(buf))
			t.Logf("Pool default size is 1024, but got capacity %d", cap(buf))
			return
		}
	}
}

// TestBufferPool_EmptyBufferPut_BugReproduction
// This test shows what happens when we put a truly empty buffer
func TestBufferPool_EmptyBufferPut_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Put an empty buffer
	emptyBuf := []byte{}
	pool.Put(emptyBuf)

	// Get a buffer
	buf := pool.Get()

	if cap(buf) < 64 {
		t.Errorf("CRITICAL BUG: After putting empty buffer, Get() returned buffer with capacity %d", cap(buf))
	}
}

// TestBufferPool_ReuseScenario_BugReproduction
// This test shows a realistic reuse scenario where the bug manifests
func TestBufferPool_ReuseScenario_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Simulate real usage pattern
	for i := 0; i < 20; i++ {
		buf := pool.Get()

		// Try to use the buffer - this would panic if capacity is 0
		if cap(buf) == 0 {
			t.Fatalf("CRITICAL BUG at iteration %d: Cannot use buffer with zero capacity", i)
		}

		// Write some data
		if cap(buf) > 0 {
			buf = append(buf, byte(i%256))
		}

		// Return to pool
		pool.Put(buf)
	}

	t.Logf("OK: All 20 iterations completed without zero-capacity buffers")
}

// TestBufferPool_ConcurrentPutGet_BugReproduction
// This test shows concurrent access issues
func TestBufferPool_ConcurrentPutGet_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 4, 8)

	bugFound := false
	var mu sync.Mutex
	done := make(chan bool, 10)

	// Launch several goroutines that put and get buffers
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Put a buffer with unexpected capacity
			buf := make([]byte, 0, id*10) // varying capacities
			pool.Put(buf)

			// Get a buffer
			gotBuf := pool.Get()
			if cap(gotBuf) < 64 {
				mu.Lock()
				bugFound = true
				mu.Unlock()
				t.Logf("Goroutine %d: CRITICAL BUG - got buffer with capacity %d", id, cap(gotBuf))
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	if bugFound {
		t.Error("Concurrent test found critical bugs in buffer pool")
	}
}

// TestBufferPool_DetailedCapacityAnalysis_BugReproduction
// This test provides detailed analysis of buffer capacities
func TestBufferPool_DetailedCapacityAnalysis_BugReproduction(t *testing.T) {
	pool := bufferpool.NewByteBufferPool(1024, 2, 8)

	// Test 1: Initial get should give us proper capacity
	buf1 := pool.Get()
	t.Logf("Test 1 - Initial get: cap=%d, len=%d", cap(buf1), len(buf1))
	if cap(buf1) < 64 {
		t.Errorf("Initial get failed: capacity %d < 64", cap(buf1))
	}

	// Test 2: Put a small buffer and see if we get it back
	smallBuf := make([]byte, 0, 1)
	pool.Put(smallBuf)

	buf2 := pool.Get()
	t.Logf("Test 2 - After putting capacity=1 buffer: got cap=%d, len=%d", cap(buf2), len(buf2))
	if cap(buf2) < 64 {
		t.Errorf("CRITICAL BUG: Got buffer with capacity %d after putting capacity=1 buffer", cap(buf2))
	}

	// Test 3: Put zero capacity buffer
	zeroBuf := make([]byte, 0, 0)
	pool.Put(zeroBuf)

	buf3 := pool.Get()
	t.Logf("Test 3 - After putting capacity=0 buffer: got cap=%d, len=%d", cap(buf3), len(buf3))
	if cap(buf3) < 64 {
		t.Errorf("CRITICAL BUG: Got buffer with capacity %d after putting capacity=0 buffer", cap(buf3))
	}

	// Test 4: Multiple puts and gets
	capacities := []int{0, 1, 10, 50, 100, 500}
	for _, capToPut := range capacities {
		buf := make([]byte, 0, capToPut)
		pool.Put(buf)
	}

	buf4 := pool.Get()
	t.Logf("Test 4 - After putting buffers with capacities %v: got cap=%d, len=%d", capacities, cap(buf4), len(buf4))
	if cap(buf4) < 64 {
		t.Errorf("CRITICAL BUG: Got buffer with capacity %d after putting varied capacities", cap(buf4))
	}
}

// TestBufferPool_DefaultPoolBehavior_BugReproduction
// This tests the default pool behavior
func TestBufferPool_DefaultPoolBehavior_BugReproduction(t *testing.T) {
	// Use the default pool
	buf1 := bufferpool.Get()
	t.Logf("Default pool - First get: cap=%d, len=%d", cap(buf1), len(buf1))

	// Put a problematic buffer
	badBuf := make([]byte, 0, 0)
	bufferpool.Put(badBuf)

	// Get again
	buf2 := bufferpool.Get()
	t.Logf("Default pool - After putting zero-capacity buffer: cap=%d, len=%d", cap(buf2), len(buf2))

	if cap(buf2) < 64 {
		t.Errorf("CRITICAL BUG: Default pool returned buffer with capacity %d", cap(buf2))
	}
}
