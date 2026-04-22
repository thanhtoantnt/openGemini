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

func TestBufferPool_SimpleRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(1, 1024*1024).Draw(t, "size")

		buf1 := bufferpool.Get()
		if cap(buf1) < 64 { // minDefaultSize
			t.Fatalf("Buffer capacity %d less than minimum expected", cap(buf1))
		}

		for i := 0; i < min(size, cap(buf1)); i++ {
			buf1 = append(buf1, byte(i%256))
		}

		bufferpool.Put(buf1)

		buf2 := bufferpool.Get()

		if cap(buf2) < 64 {
			t.Fatalf("Reused buffer capacity %d less than minimum expected", cap(buf2))
		}

		bufferpool.Put(buf2)
	})
}

func TestBufferPool_CustomPool(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defaultSize := rapid.Uint64Range(64, 1024*1024).Draw(t, "defaultSize")
		localCacheNum := rapid.IntRange(1, 8).Draw(t, "localCacheNum")

		pool := bufferpool.NewByteBufferPool(defaultSize, localCacheNum, 8)

		size := rapid.IntRange(1, int(defaultSize)).Draw(t, "size")
		buf := pool.Get()

		if cap(buf) < 64 {
			t.Fatalf("Buffer capacity %d less than minimum expected", cap(buf))
		}

		for i := 0; i < min(size, cap(buf)); i++ {
			buf = append(buf, byte(i%256))
		}

		pool.Put(buf)
	})
}

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

func TestBufferPool_ConcurrentGetPut(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numOps := rapid.IntRange(10, 100).Draw(t, "numOps")
		pool := bufferpool.NewByteBufferPool(1024, 4, 8)

		done := make(chan bool, numOps)

		for i := 0; i < numOps; i++ {
			go func() {
				buf := pool.Get()
				if cap(buf) < 64 {
					t.Fatalf("Buffer capacity %d less than minimum expected", cap(buf))
				}
				buf = append(buf, byte(1))
				pool.Put(buf)
				done <- true
			}()
		}

		for i := 0; i < numOps; i++ {
			<-done
		}
	})
}

func TestBufferPool_SizeInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(64, 1024*1024).Draw(t, "size")

		buf := bufferpool.Get()

		if len(buf) != 0 {
			t.Fatalf("New buffer should be empty: got %d", len(buf))
		}

		if cap(buf) < 64 {
			t.Fatalf("Buffer capacity %d less than minimum", cap(buf))
		}

		if cap(buf) > 32*1024*1024 {
			t.Fatalf("Buffer capacity %d exceeds maximum local cache size", cap(buf))
		}

		data := make([]byte, min(size, cap(buf)))
		buf = append(buf, data...)

		bufferpool.Put(buf)

		buf2 := bufferpool.Get()
		if len(buf2) != 0 {
			t.Fatalf("Reused buffer should be empty: got %d", len(buf2))
		}

		bufferpool.Put(buf2)
	})
}

func TestBufferPool_DefaultPool(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		buf := bufferpool.Get()

		if len(buf) != 0 {
			t.Fatalf("Default pool buffer should be empty: got %d", len(buf))
		}

		if cap(buf) < 64 {
			t.Fatalf("Default pool buffer capacity %d less than minimum", cap(buf))
		}

		bufferpool.Put(buf)

		buf2 := bufferpool.Get()
		if len(buf2) != 0 {
			t.Fatalf("Default pool reused buffer should be empty: got %d", len(buf2))
		}

		bufferpool.Put(buf2)
	})
}

func TestBufferPool_LargeBuffers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(32*1024*1024, 64*1024*1024).Draw(t, "size")

		buf := make([]byte, 0, size)
		for i := 0; i < min(size, cap(buf)); i++ {
			buf = append(buf, byte(i%256))
		}

		bufferpool.Put(buf)

		buf2 := bufferpool.Get()

		if len(buf2) != 0 {
			t.Fatalf("Large buffer after put should be empty: got %d", len(buf2))
		}

		bufferpool.Put(buf2)
	})
}

func TestBufferPool_ResizeZero(t *testing.T) {
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

func TestBufferPool_ResizeSmaller(t *testing.T) {
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
				t.Fatalf("Resize to smaller corrupted data at position %d: got %d, want %d", i, resized[i], byte(i))
			}
		}
	})
}

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
				t.Fatalf("Resize to same corrupted data at position %d: got %d, want %d", i, resized[i], byte(i))
			}
		}
	})
}

func TestBufferPool_MultiplePools(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pool1 := bufferpool.NewByteBufferPool(512, 2, 4)
		pool2 := bufferpool.NewByteBufferPool(2048, 2, 4)

		buf1 := pool1.Get()
		buf2 := pool2.Get()

		if cap(buf1) < 64 || cap(buf2) < 64 {
			t.Fatalf("Both pools should provide buffers with minimum capacity")
		}

		pool1.Put(buf1)
		pool2.Put(buf2)

		buf1Again := pool1.Get()
		buf2Again := pool2.Get()

		if len(buf1Again) != 0 || len(buf2Again) != 0 {
			t.Fatalf("Reused buffers should be empty")
		}

		pool1.Put(buf1Again)
		pool2.Put(buf2Again)
	})
}
