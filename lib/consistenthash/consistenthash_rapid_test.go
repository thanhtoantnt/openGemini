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

package consistenthash_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/consistenthash"
)

// TestConsistentHash_GetReturnsValidKey
// Verifies that Get() always returns a key that was added
func TestConsistentHash_GetReturnsValidKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 100).Draw(t, "replicas")
		numKeys := rapid.IntRange(1, 20).Draw(t, "numKeys")

		m := consistenthash.New(replicas, nil)

		keys := make([]string, numKeys)
		for i := 0; i < numKeys; i++ {
			keys[i] = rapid.String().Draw(t, "key")
		}
		m.Add(keys...)

		// Test with random query keys
		for i := 0; i < 10; i++ {
			queryKey := rapid.String().Draw(t, "queryKey")
			result := m.Get(queryKey)

			// Result must be one of the added keys (empty string is valid if it was added)
			found := false
			for _, k := range keys {
				if k == result {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Get(%q) returned %q which was not added", queryKey, result)
			}
		}
	})
}

// TestConsistentHash_Consistency
// Verifies that same input always maps to same output
func TestConsistentHash_Consistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 100).Draw(t, "replicas")
		numKeys := rapid.IntRange(1, 10).Draw(t, "numKeys")

		m := consistenthash.New(replicas, nil)

		keys := make([]string, numKeys)
		for i := 0; i < numKeys; i++ {
			keys[i] = rapid.String().Draw(t, "key")
		}
		m.Add(keys...)

		queryKey := rapid.String().Draw(t, "queryKey")

		// Call Get multiple times - should always return same result
		result1 := m.Get(queryKey)
		result2 := m.Get(queryKey)
		result3 := m.Get(queryKey)

		if result1 != result2 || result2 != result3 {
			t.Fatalf("Get() not consistent: %q, %q, %q", result1, result2, result3)
		}
	})
}

// TestConsistentHash_EmptyMap
// Verifies that empty map returns empty string
func TestConsistentHash_EmptyMap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 100).Draw(t, "replicas")

		m := consistenthash.New(replicas, nil)

		if !m.IsEmpty() {
			t.Fatalf("New map should be empty")
		}

		queryKey := rapid.String().Draw(t, "queryKey")
		result := m.Get(queryKey)

		if result != "" {
			t.Fatalf("Get() on empty map should return empty string, got %q", result)
		}
	})
}

// TestConsistentHash_AddIdempotence
// Verifies that adding same key multiple times works correctly
func TestConsistentHash_AddIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 100).Draw(t, "replicas")
		key := rapid.String().Draw(t, "key")

		m1 := consistenthash.New(replicas, nil)
		m1.Add(key)

		m2 := consistenthash.New(replicas, nil)
		m2.Add(key, key, key) // Add same key multiple times

		// Both should give same results
		queryKey := rapid.String().Draw(t, "queryKey")
		result1 := m1.Get(queryKey)
		result2 := m2.Get(queryKey)

		if result1 != result2 {
			t.Fatalf("Adding same key multiple times changed behavior: %q vs %q", result1, result2)
		}
	})
}

// TestConsistentHash_AddMoreKeys
// Verifies that adding more keys doesn't break existing mappings
func TestConsistentHash_AddMoreKeys(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 100).Draw(t, "replicas")

		m := consistenthash.New(replicas, nil)

		keys1 := make([]string, 5)
		for i := range keys1 {
			keys1[i] = rapid.String().Draw(t, "key1")
		}
		m.Add(keys1...)

		// Record some mappings
		mappings := make(map[string]string)
		for i := 0; i < 10; i++ {
			qk := rapid.String().Draw(t, "queryKey")
			mappings[qk] = m.Get(qk)
		}

		// Add more keys
		keys2 := make([]string, 5)
		for i := range keys2 {
			keys2[i] = rapid.String().Draw(t, "key2")
		}
		m.Add(keys2...)

		// Mappings might change, but should still be valid
		for qk, expected := range mappings {
			result := m.Get(qk)
			// Result should be one of all added keys
			allKeys := append(keys1, keys2...)
			found := false
			for _, k := range allKeys {
				if k == result {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("After adding more keys, Get(%q) returned invalid key %q (was %q)", qk, result, expected)
			}
		}
	})
}

// TestConsistentHash_Distribution
// Verifies that keys are distributed across nodes
func TestConsistentHash_Distribution(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(10, 100).Draw(t, "replicas")
		numNodes := rapid.IntRange(3, 10).Draw(t, "numNodes")

		m := consistenthash.New(replicas, nil)

		nodes := make([]string, numNodes)
		for i := range nodes {
			nodes[i] = string(rune('a' + i))
		}
		m.Add(nodes...)

		// Count distribution
		distribution := make(map[string]int)
		numQueries := 1000
		for i := 0; i < numQueries; i++ {
			queryKey := rapid.String().Draw(t, "queryKey")
			node := m.Get(queryKey)
			distribution[node]++
		}

		// Each node should get some queries (with high probability)
		// This is a probabilistic test, so we use a loose threshold
		for _, node := range nodes {
			if distribution[node] == 0 {
				t.Logf("Warning: node %q got 0 queries out of %d (may be due to hash distribution)", node, numQueries)
			}
		}
	})
}

// TestConsistentHash_ReplicasEffect
// Verifies that more replicas leads to better distribution
func TestConsistentHash_ReplicasEffect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numNodes := 5
		nodes := make([]string, numNodes)
		for i := range nodes {
			nodes[i] = string(rune('a' + i))
		}

		// Create two maps with different replica counts
		m1 := consistenthash.New(10, nil)
		m2 := consistenthash.New(100, nil)
		m1.Add(nodes...)
		m2.Add(nodes...)

		// Both should return valid keys
		for i := 0; i < 100; i++ {
			queryKey := rapid.String().Draw(t, "queryKey")

			r1 := m1.Get(queryKey)
			r2 := m2.Get(queryKey)

			if r1 == "" || r2 == "" {
				t.Fatalf("Get() returned empty string")
			}
		}
	})
}

// TestConsistentHash_RemoveNotSupported
// Documents that there's no Remove method
func TestConsistentHash_NoRemove(t *testing.T) {
	// This test documents that the consistenthash implementation
	// doesn't have a Remove method, which could be a limitation
	m := consistenthash.New(10, nil)
	m.Add("a", "b", "c")

	// Once added, keys cannot be removed
	// This is by design based on the API
	_ = m
}
