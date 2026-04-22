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

package bloomfilter_test

import (
	"testing"

	"github.com/openGemini/openGemini/lib/bloomfilter"
)

// TestBloomFilter_BoundaryPanic_Version0
// Demonstrates the bug where Version 0 panics with insufficient buffer size
// BUG: Add() panics when offset+8 exceeds buffer capacity
func TestBloomFilter_BoundaryPanic_Version0(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Version 0 panics with insufficient buffer size")
			t.Logf("  Panic: %v", r)
			t.Logf("  Input: version=0, size=8, hash=0x2000000000000")
			t.Logf("  Offset calculation: hash >> 49 = 1")
			t.Logf("  Required bytes: offset+8 = 9")
			t.Logf("  Available bytes: 8")
			t.Logf("  Result: slice bounds out of range [:9] with capacity 8")
		}
	}()

	bf := bloomfilter.DefaultOneHitBloomFilter(0, 8)
	hash := uint64(0x2000000000000)
	bf.Add(hash)

	t.Errorf("Expected panic but Add() succeeded")
}

// TestBloomFilter_BoundaryPanic_Version2
// Demonstrates the bug where Version 2 panics with insufficient buffer size
// BUG: Add() panics when offset+8 exceeds buffer capacity
func TestBloomFilter_BoundaryPanic_Version2(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Version 2 panics with insufficient buffer size")
			t.Logf("  Panic: %v", r)
			t.Logf("  Input: version=2, size=8, hash=0x400000000000")
			t.Logf("  Offset calculation: hash >> 46 = 1")
			t.Logf("  Required bytes: offset+8 = 9")
			t.Logf("  Available bytes: 8")
			t.Logf("  Result: slice bounds out of range [:9] with capacity 8")
		}
	}()

	bf := bloomfilter.DefaultOneHitBloomFilter(2, 8)
	hash := uint64(0x400000000000)
	bf.Add(hash)

	t.Errorf("Expected panic but Add() succeeded")
}

// TestBloomFilter_BoundaryPanic_Version3
// Demonstrates the bug where Version 3 panics with insufficient buffer size
// BUG: Add() panics when offset+8 exceeds buffer capacity
func TestBloomFilter_BoundaryPanic_Version3(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Version 3 panics with insufficient buffer size")
			t.Logf("  Panic: %v", r)
			t.Logf("  Input: version=3, size=8, hash=0x400000000000")
			t.Logf("  Offset calculation: hash >> 46 = 1")
			t.Logf("  Required bytes: offset+8 = 9")
			t.Logf("  Available bytes: 8")
			t.Logf("  Result: slice bounds out of range [:9] with capacity 8")
		}
	}()

	bf := bloomfilter.DefaultOneHitBloomFilter(3, 8)
	hash := uint64(0x400000000000)
	bf.Add(hash)

	t.Errorf("Expected panic but Add() succeeded")
}

// TestBloomFilter_HitPanic_Version0
// Demonstrates that Hit() also panics with insufficient buffer size
func TestBloomFilter_HitPanic_Version0(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Hit() also panics with insufficient buffer size")
			t.Logf("  Panic: %v", r)
		}
	}()

	bf := bloomfilter.DefaultOneHitBloomFilter(0, 8)
	hash := uint64(0x2000000000000)
	bf.Hit(hash)

	t.Errorf("Expected panic but Hit() succeeded")
}

// TestBloomFilter_MinimumRequiredSize_Version0
// Documents the minimum required size for Version 0
func TestBloomFilter_MinimumRequiredSize_Version0(t *testing.T) {
	// Maximum offset for V0: hash >> 49 = 2^15 - 1 = 32767
	// Minimum required size: 32767 + 8 = 32775 bytes

	minSize := int64(32775)
	bf := bloomfilter.DefaultOneHitBloomFilter(0, minSize)

	// Test with maximum hash that produces maximum offset
	// Max offset = 32767, so hash = 32767 << 49
	maxHash := uint64(0x7FFF) << 49 // This gives offset = 32767

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic with minimum required size %d: %v", minSize, r)
			}
		}()
		bf.Add(maxHash)
		t.Logf("OK: Version 0 works with minimum size %d", minSize)
	}()
}

// TestBloomFilter_MinimumRequiredSize_Version2
// Documents the minimum required size for Version 2
func TestBloomFilter_MinimumRequiredSize_Version2(t *testing.T) {
	// Maximum offset for V2: hash >> 46 = 2^18 - 1 = 262143
	// Minimum required size: 262143 + 8 = 262151 bytes

	minSize := int64(262151)
	bf := bloomfilter.DefaultOneHitBloomFilter(2, minSize)

	// Test with maximum hash that produces maximum offset
	maxHash := uint64(0x3FFFF) << 46 // This gives offset = 262143

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic with minimum required size %d: %v", minSize, r)
			}
		}()
		bf.Add(maxHash)
		t.Logf("OK: Version 2 works with minimum size %d", minSize)
	}()
}

// TestBloomFilter_ProductionSizes
// Verifies that production sizes are sufficient (but barely)
func TestBloomFilter_ProductionSizes(t *testing.T) {
	testCases := []struct {
		version uint32
		size    int64
		desc    string
	}{
		{0, 32*1024 + 64, "Production V0 size"},
		{2, 256*1024 + 64, "Production V2 size"},
		{3, 256*1024 + 64, "Production V3 size"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			bf := bloomfilter.DefaultOneHitBloomFilter(tc.version, tc.size)

			// Test with various hash values
			testHashes := []uint64{
				0,                  // Minimum hash
				0xFFFFFFFFFFFFFFFF, // Maximum hash
				0x8000000000000000, // High bit set
				12345,              // Small hash
				0xDEADBEEFCAFEBABE, // Random-looking hash
			}

			for _, hash := range testHashes {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("Panic with hash 0x%x: %v", hash, r)
						}
					}()
					bf.Add(hash)
					if !bf.Hit(hash) {
						t.Errorf("Hash 0x%x not found after Add", hash)
					}
				}()
			}
		})
	}
}

// TestBloomFilter_OffByOnePanic
// Demonstrates that even one byte less than required causes panic
func TestBloomFilter_OffByOnePanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: Off-by-one in buffer size causes panic")
			t.Logf("  Minimum required: 32775 bytes")
			t.Logf("  Provided: 32774 bytes (one less)")
			t.Logf("  Panic: %v", r)
		}
	}()

	// Try with one byte less than minimum required
	bf := bloomfilter.DefaultOneHitBloomFilter(0, 32774)

	// Hash that produces maximum offset
	maxHash := uint64(0x7FFF) << 49
	bf.Add(maxHash)

	t.Errorf("Expected panic but Add() succeeded")
}
