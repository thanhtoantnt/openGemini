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
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/bufferpool"
)

// TestBufferPool_CapacityPreservation
// Verifies that the pool preserves buffer capacities (core API contract)
func TestBufferPool_CapacityPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(0, 1024*1024).Draw(t, "capacity")
		pool := bufferpool.NewByteBufferPool(1024, 2, 8)

		// Put buffer with specific capacity
		buf := make([]byte, 0, capacity)
		pool.Put(buf)

		// Get should return buffer with same capacity
		buf2 := pool.Get()

		if cap(buf2) != capacity {
			t.Fatalf("Capacity not preserved: put cap=%d, got cap=%d", capacity, cap(buf2))
		}
	})
}

// TestBufferPool_LengthReset
// Verifies that Put always resets buffer length to 0
func TestBufferPool_LengthReset(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, 1024*1024).Draw(t, "capacity")
		dataLen := rapid.IntRange(0, capacity).Draw(t, "dataLen")
		pool := bufferpool.NewByteBufferPool(1024, 2, 8)

		// Create buffer with data
		buf := make([]byte, 0, capacity)
		for i := 0; i < dataLen; i++ {
			buf = append(buf, byte(i%256))
		}

		pool.Put(buf)
		buf2 := pool.Get()

		// Length should always be 0 after Put
		if len(buf2) != 0 {
			t.Fatalf("Length not reset: expected 0, got %d", len(buf2))
		}
	})
}

// TestBufferPool_ResizePreservesData
// Verifies that Resize preserves existing data
func TestBufferPool_ResizePreservesData(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		originalSize := rapid.IntRange(10, 100).Draw(t, "originalSize")
		newSize := rapid.IntRange(originalSize, originalSize*10).Draw(t, "newSize")

		buf := make([]byte, 0, originalSize)
		for i := 0; i < originalSize; i++ {
			buf = append(buf, byte(i))
		}

		resized := bufferpool.Resize(buf, newSize)

		if len(resized) < originalSize {
			t.Fatalf("Resize truncated data: got %d, want %d", len(resized), originalSize)
		}

		for i := 0; i < originalSize; i++ {
			if resized[i] != byte(i) {
				t.Fatalf("Resize corrupted data at position %d: got %d, want %d", i, resized[i], byte(i))
			}
		}

		if cap(resized) < newSize {
			t.Fatalf("Resize capacity %d less than requested size %d", cap(resized), newSize)
		}
	})
}

// TestBufferPool_ResizeToSmaller
// Verifies Resize behavior when shrinking
func TestBufferPool_ResizeToSmaller(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		originalSize := rapid.IntRange(100, 200).Draw(t, "originalSize")
		newSize := rapid.IntRange(10, originalSize-10).Draw(t, "newSize")

		buf := make([]byte, originalSize)
		for i := 0; i < originalSize; i++ {
			buf[i] = byte(i)
		}

		resized := bufferpool.Resize(buf, newSize)

		if len(resized) != newSize {
			t.Fatalf("Resize to smaller size: got %d, want %d", len(resized), newSize)
		}

		for i := 0; i < newSize; i++ {
			if resized[i] != byte(i) {
				t.Fatalf("Resize to smaller corrupted data at position %d", i)
			}
		}
	})
}

// TestBufferPool_ResizeToZero
// Verifies Resize to zero length
func TestBufferPool_ResizeToZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		originalSize := rapid.IntRange(10, 100).Draw(t, "originalSize")

		buf := make([]byte, originalSize)
		for i := 0; i < originalSize; i++ {
			buf[i] = byte(i)
		}

		resized := bufferpool.Resize(buf, 0)

		if len(resized) != 0 {
			t.Fatalf("Resize to zero should produce empty buffer: got %d", len(resized))
		}
	})
}

// TestBufferPool_ResizeSame
// Verifies Resize to same size
func TestBufferPool_ResizeSame(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(10, 100).Draw(t, "size")

		buf := make([]byte, size)
		for i := 0; i < size; i++ {
			buf[i] = byte(i)
		}

		resized := bufferpool.Resize(buf, size)

		if len(resized) != size {
			t.Fatalf("Resize to same size: got %d, want %d", len(resized), size)
		}

		for i := 0; i < size; i++ {
			if resized[i] != byte(i) {
				t.Fatalf("Resize to same corrupted data at position %d", i)
			}
		}
	})
}

// TestBufferPool_MultiplePools
// Verifies that different pools operate independently
func TestBufferPool_MultiplePools(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cap1 := rapid.IntRange(10, 100).Draw(t, "cap1")
		cap2 := rapid.IntRange(10, 100).Draw(t, "cap2")

		pool1 := bufferpool.NewByteBufferPool(512, 2, 4)
		pool2 := bufferpool.NewByteBufferPool(2048, 2, 4)

		buf1 := make([]byte, 0, cap1)
		buf2 := make([]byte, 0, cap2)

		pool1.Put(buf1)
		pool2.Put(buf2)

		buf1Out := pool1.Get()
		buf2Out := pool2.Get()

		if cap(buf1Out) != cap1 {
			t.Fatalf("Pool1 capacity not preserved: expected %d, got %d", cap1, cap(buf1Out))
		}
		if cap(buf2Out) != cap2 {
			t.Fatalf("Pool2 capacity not preserved: expected %d, got %d", cap2, cap(buf2Out))
		}
	})
}

// TestBufferPool_DefaultPool
// Verifies default pool behavior
func TestBufferPool_DefaultPool(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(0, 1024*1024).Draw(t, "capacity")

		buf := make([]byte, 0, capacity)
		bufferpool.Put(buf)

		buf2 := bufferpool.Get()

		if len(buf2) != 0 {
			t.Fatalf("Default pool buffer should be empty: got %d", len(buf2))
		}

		if cap(buf2) != capacity {
			t.Fatalf("Default pool capacity not preserved: expected %d, got %d", capacity, cap(buf2))
		}
	})
}

// TestBufferPool_PutGetIdempotence
// Verifies that put/get cycles preserve buffer properties
func TestBufferPool_PutGetIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(1, 1024).Draw(t, "capacity")
		pool := bufferpool.NewByteBufferPool(1024, 2, 8)

		// First cycle
		buf1 := make([]byte, 0, capacity)
		pool.Put(buf1)
		buf1Out := pool.Get()

		// Second cycle
		pool.Put(buf1Out)
		buf2Out := pool.Get()

		// Both should have same capacity
		if cap(buf1Out) != cap(buf2Out) {
			t.Fatalf("Idempotence violated: first cap=%d, second cap=%d", cap(buf1Out), cap(buf2Out))
		}

		// Both should have length 0
		if len(buf1Out) != 0 || len(buf2Out) != 0 {
			t.Fatalf("Buffers should have length 0")
		}
	})
}

// TestBufferPool_LargeBuffers
// Verifies handling of large buffers (> MaxLocalCacheSize)
func TestBufferPool_LargeBuffers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buffers larger than MaxLocalCacheSize (32MB) go to underlying pool
		size := rapid.IntRange(32*1024*1024+1, 64*1024*1024).Draw(t, "size")

		buf := make([]byte, 0, size)
		bufferpool.Put(buf)

		buf2 := bufferpool.Get()

		if len(buf2) != 0 {
			t.Fatalf("Large buffer should be empty after Put: got %d", len(buf2))
		}

		if cap(buf2) != size {
			t.Fatalf("Large buffer capacity not preserved: expected %d, got %d", size, cap(buf2))
		}
	})
}
