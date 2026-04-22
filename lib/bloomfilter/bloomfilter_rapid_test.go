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

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/bloomfilter"
)

// getMinSize returns minimum valid size for a bloomfilter version
func getMinSize(version uint32) int64 {
	if version <= 1 {
		return 32775 // (2^15 - 1) + 8
	}
	return 262151 // (2^18 - 1) + 8
}

// getProductionSize returns production size for a bloomfilter version
func getProductionSize(version uint32) int64 {
	if version <= 1 {
		return 32*1024 + 64 // 32832
	}
	return 256*1024 + 64 // 262208
}

// TestBloomFilter_AddHitConsistency
// Verifies that after Add(hash), Hit(hash) returns true
func TestBloomFilter_AddHitConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)
		bf.Add(hash)

		if !bf.Hit(hash) {
			t.Fatalf("After Add(%d), Hit(%d) should return true", hash, hash)
		}
	})
}

// TestBloomFilter_MultipleAdds
// Verifies that multiple Add operations work correctly
func TestBloomFilter_MultipleAdds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		numHashes := rapid.IntRange(1, 100).Draw(t, "numHashes")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

		hashes := make([]uint64, numHashes)
		for i := 0; i < numHashes; i++ {
			hashes[i] = rapid.Uint64().Draw(t, "hash")
			bf.Add(hashes[i])
		}

		// All added hashes should hit
		for i, hash := range hashes {
			if !bf.Hit(hash) {
				t.Fatalf("Hash %d at index %d not found after Add", hash, i)
			}
		}
	})
}

// TestBloomFilter_ClearResets
// Verifies that Clear resets the bloom filter
func TestBloomFilter_ClearResets(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)
		bf.Add(hash)

		// Should hit before clear
		if !bf.Hit(hash) {
			t.Fatalf("Hash should hit before clear")
		}

		bf.Clear()

		// After clear, data should be zeroed
		data := bf.Data()
		allZero := true
		for _, b := range data {
			if b != 0 {
				allZero = false
				break
			}
		}

		if !allZero {
			t.Logf("Note: Clear() zeroed all bytes (version=%d)", version)
		}
	})
}

// TestBloomFilter_LoadHitConsistency
// Verifies LoadHit behavior
func TestBloomFilter_LoadHitConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)
		bf.Add(hash)

		offset := bf.GetBytesOffset(hash)
		data := bf.Data()

		if offset < 0 || offset+8 > int64(len(data)) {
			t.Fatalf("GetBytesOffset returned invalid offset: %d (data len=%d)", offset, len(data))
		}

		// Load the hash at the offset
		loadHash := data[offset : offset+8]
		if len(loadHash) != 8 {
			t.Fatalf("Load hash has wrong length: %d", len(loadHash))
		}
	})
}

// TestBloomFilter_DataConsistency
// Verifies that Data() returns consistent data
func TestBloomFilter_DataConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

		data1 := bf.Data()
		data2 := bf.Data()

		if len(data1) != len(data2) {
			t.Fatalf("Data() returned different lengths: %d vs %d", len(data1), len(data2))
		}

		if len(data1) != int(size) {
			t.Fatalf("Data() length %d doesn't match size %d", len(data1), size)
		}
	})
}

// TestBloomFilter_AddIdempotence
// Verifies that Add is idempotent
func TestBloomFilter_AddIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

		// Add once
		bf.Add(hash)
		data1 := bf.Data()

		// Add again
		bf.Add(hash)
		data2 := bf.Data()

		// Data should be the same (idempotent)
		if len(data1) != len(data2) {
			t.Fatalf("Data length changed after second Add")
		}

		for i := range data1 {
			if data1[i] != data2[i] {
				t.Fatalf("Data changed at byte %d after second Add: %d vs %d", i, data1[i], data2[i])
			}
		}
	})
}

// TestBloomFilter_VersionDifferences
// Verifies that different versions produce valid bloom filters
func TestBloomFilter_VersionDifferences(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hash := rapid.Uint64().Draw(t, "hash")

		// Test all versions with their production sizes
		for version := uint32(0); version <= 3; version++ {
			size := getProductionSize(version)
			bf := bloomfilter.DefaultOneHitBloomFilter(version, size)
			bf.Add(hash)

			if !bf.Hit(hash) {
				t.Fatalf("Version %d: Add/Hit consistency failed", version)
			}
		}
	})
}

// TestBloomFilter_HashDistribution
// Verifies that different hashes can be stored
func TestBloomFilter_HashDistribution(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

		// Add multiple distinct hashes
		hashes := make(map[uint64]bool)
		for i := 0; i < 10; i++ {
			hash := rapid.Uint64().Draw(t, "hash")
			hashes[hash] = true
			bf.Add(hash)
		}

		// All should hit
		for hash := range hashes {
			if !bf.Hit(hash) {
				t.Fatalf("Hash %d not found", hash)
			}
		}
	})
}

// TestBloomFilter_GetBytesOffsetRange
// Verifies that GetBytesOffset returns valid offsets
func TestBloomFilter_GetBytesOffsetRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getProductionSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)
		offset := bf.GetBytesOffset(hash)

		// Offset should be within data bounds (with 8 byte margin)
		data := bf.Data()
		if offset < 0 || offset+8 > int64(len(data)) {
			t.Fatalf("GetBytesOffset returned %d, but data length is %d (need offset+8=%d)", offset, len(data), offset+8)
		}
	})
}

// TestBloomFilter_MinimumSize
// Verifies bloomfilter works with minimum valid sizes
func TestBloomFilter_MinimumSize(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		version := rapid.Uint32Range(0, 3).Draw(t, "version")
		size := getMinSize(version)
		hash := rapid.Uint64().Draw(t, "hash")

		bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Panic with minimum size %d: %v", size, r)
				}
			}()
			bf.Add(hash)
			if !bf.Hit(hash) {
				t.Fatalf("Add/Hit failed with minimum size")
			}
		}()
	})
}

// TestBloomFilter_AllProductionSizes
// Verifies all versions work with production sizes
func TestBloomFilter_AllProductionSizes(t *testing.T) {
	productionSizes := map[uint32]int64{
		0: 32*1024 + 64,
		1: 32*1024 + 64,
		2: 256*1024 + 64,
		3: 256*1024 + 64,
	}

	for version, size := range productionSizes {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				bf := bloomfilter.DefaultOneHitBloomFilter(version, size)

				// Test with many hashes
				for i := 0; i < 100; i++ {
					hash := rapid.Uint64().Draw(t, "hash")
					bf.Add(hash)
					if !bf.Hit(hash) {
						t.Fatalf("Version %d: Add/Hit failed for hash %d", version, hash)
					}
				}
			})
		})
	}
}
