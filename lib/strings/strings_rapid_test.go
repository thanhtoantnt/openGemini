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

package strings

import (
	"sort"
	stdstrings "strings"
	"testing"

	"pgregory.net/rapid"
)

func TestUnionSlice_NoDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.SliceOfN(rapid.String(), 1, 1000).Draw(t, "input")
		result := UnionSlice(input)

		if len(result) > len(input) {
			t.Fatalf("UnionSlice should not increase length: got %d, input %d", len(result), len(input))
		}

		seen := make(map[string]bool)
		for _, s := range result {
			if seen[s] {
				t.Fatalf("UnionSlice contains duplicate: %s", s)
			}
			seen[s] = true
		}
	})
}

func TestUnionSlice_AllOriginalElementsPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.SliceOfN(rapid.String(), 1, 1000).Draw(t, "input")
		result := UnionSlice(input)

		inputSet := make(map[string]bool)
		for _, s := range input {
			inputSet[s] = true
		}

		for _, s := range result {
			if !inputSet[s] {
				t.Fatalf("UnionSlice contains element not in input: %s", s)
			}
		}
	})
}

func TestUnionSlice_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.SliceOfN(rapid.String(), 1, 1000).Draw(t, "input")

		firstResult := UnionSlice(input)
		secondResult := UnionSlice(firstResult)

		if len(firstResult) != len(secondResult) {
			t.Fatalf("UnionSlice should be idempotent: first=%d, second=%d", len(firstResult), len(secondResult))
		}

		sort.Strings(firstResult)
		sort.Strings(secondResult)

		if !SortIsEqual(firstResult, secondResult) {
			t.Fatalf("UnionSlice should be idempotent")
		}
	})
}

func TestUnionSlice_EmptyInput(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		empty := []string{}
		result := UnionSlice(empty)

		if len(result) != 0 {
			t.Fatalf("UnionSlice of empty should be empty: got %d elements", len(result))
		}
	})
}

func TestUnionSlice_SingleElement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		single := rapid.String().Draw(t, "single")
		input := []string{single}

		result := UnionSlice(input)

		if len(result) != 1 || result[0] != single {
			t.Fatalf("UnionSlice of single element should preserve it")
		}
	})
}

func TestUnionSlice_AlreadyUnique(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		uniqueInput := make(map[string]bool)
		numElements := rapid.IntRange(1, 100).Draw(t, "numElements")

		input := make([]string, 0, numElements)
		for len(uniqueInput) < numElements {
			s := rapid.String().Draw(t, "string")
			if !uniqueInput[s] {
				uniqueInput[s] = true
				input = append(input, s)
			}
		}

		result := UnionSlice(input)

		if len(result) != len(input) {
			t.Fatalf("UnionSlice should not change length of already unique input: got %d, want %d",
				len(result), len(input))
		}
	})
}

func TestUnionSlice_AllDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "string")
		count := rapid.IntRange(1, 100).Draw(t, "count")

		input := make([]string, count)
		for i := range input {
			input[i] = s
		}

		result := UnionSlice(input)

		if len(result) != 1 || result[0] != s {
			t.Fatalf("UnionSlice of all duplicates should return single element")
		}
	})
}

func TestUnionSlice_MixedDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		uniqueStrings := make(map[string]bool)
		numUnique := rapid.IntRange(1, 50).Draw(t, "numUnique")

		uniqueInput := make([]string, 0, numUnique)
		for len(uniqueStrings) < numUnique {
			s := rapid.String().Draw(t, "string")
			if !uniqueStrings[s] {
				uniqueStrings[s] = true
				uniqueInput = append(uniqueInput, s)
			}
		}

		input := make([]string, 0)
		for _, s := range uniqueInput {
			dupCount := rapid.IntRange(1, 5).Draw(t, "dupCount")
			for i := 0; i < dupCount; i++ {
				input = append(input, s)
			}
		}

		result := UnionSlice(input)

		if len(result) != numUnique {
			t.Fatalf("UnionSlice should return exactly unique elements: got %d, want %d",
				len(result), numUnique)
		}

		resultSet := make(map[string]bool)
		for _, s := range result {
			if !uniqueStrings[s] {
				t.Fatalf("UnionSlice returned unexpected element: %s", s)
			}
			resultSet[s] = true
		}

		for s := range uniqueStrings {
			if !resultSet[s] {
				t.Fatalf("UnionSlice missed unique element: %s", s)
			}
		}
	})
}

func TestSortIsEqual_Sorted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.SliceOfN(rapid.String(), 0, 100).Draw(t, "a")
		b := make([]string, len(a))
		copy(b, a)

		sort.Strings(a)
		sort.Strings(b)

		if !SortIsEqual(a, b) {
			t.Fatalf("SortIsEqual should return true for identical sorted arrays")
		}
	})
}

func TestSortIsEqual_DifferentLengths(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.SliceOfN(rapid.String(), 0, 100).Draw(t, "a")
		b := rapid.SliceOfN(rapid.String(), 0, 100).Draw(t, "b")

		sort.Strings(a)
		sort.Strings(b)

		if len(a) != len(b) {
			if SortIsEqual(a, b) {
				t.Fatalf("SortIsEqual should return false for different lengths")
			}
		}
	})
}

func TestSortIsEqual_DifferentContent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.SliceOfN(rapid.String(), 1, 100).Draw(t, "a")
		b := rapid.SliceOfN(rapid.String(), 1, 100).Draw(t, "b")

		sort.Strings(a)
		sort.Strings(b)

		if len(a) == len(b) {
			allEqual := true
			for i := range a {
				if a[i] != b[i] {
					allEqual = false
					break
				}
			}

			if allEqual && SortIsEqual(a, b) {

			} else if !allEqual && SortIsEqual(a, b) {
				t.Fatalf("SortIsEqual should return false for different content")
			}
		}
	})
}

func TestSortIsEqual_Empty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := []string{}
		b := []string{}

		if !SortIsEqual(a, b) {
			t.Fatalf("SortIsEqual should return true for two empty arrays")
		}
	})
}

func TestSortIsEqual_SingleElement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "string")
		a := []string{s}
		b := []string{s}

		if !SortIsEqual(a, b) {
			t.Fatalf("SortIsEqual should return true for identical single elements")
		}
	})
}

func TestSortIsEqual_SameElementsDifferentOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elements := rapid.SliceOfN(rapid.String(), 2, 50).Draw(t, "elements")

		a := make([]string, len(elements))
		b := make([]string, len(elements))
		copy(a, elements)
		copy(b, elements)

		sort.Strings(a)
		sort.Strings(b)

		if !SortIsEqual(a, b) {
			t.Fatalf("SortIsEqual should return true for same elements in different order (after sorting)")
		}
	})
}

func TestContainsInterface_String(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		str := interface{}(rapid.String().Draw(t, "str"))
		sub := rapid.String().Draw(t, "sub")

		result := ContainsInterface(str, sub)

		expectedContains := false
		if s, ok := str.(string); ok {
			expectedContains = stdstrings.Contains(s, sub)
		}

		if result != expectedContains {
			t.Fatalf("ContainsInterface result mismatch: str=%v, sub=%v, got=%v, want=%v", str, sub, result, expectedContains)
		}
	})
}

func TestContainsInterface_NonString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nonString := rapid.SampledFrom([]interface{}{
			123,
			3.14,
			true,
			[]int{1, 2, 3},
			map[string]int{"key": 1},
		}).Draw(t, "nonString")

		sub := "substring"

		result := ContainsInterface(nonString, sub)

		if result {
			t.Fatalf("ContainsInterface should return false for non-string interface")
		}
	})
}

func TestEqualInterface_String(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		str := interface{}(rapid.String().Draw(t, "str"))
		compare := rapid.String().Draw(t, "compare")

		result := EqualInterface(str, compare)

		expectedEqual := false
		if s, ok := str.(string); ok {
			expectedEqual = (s == compare)
		}

		if result != expectedEqual {
			t.Fatalf("EqualInterface result mismatch")
		}
	})
}

func TestEqualInterface_NonString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nonString := rapid.SampledFrom([]interface{}{
			123,
			3.14,
			true,
			[]int{1, 2, 3},
			map[string]int{"key": 1},
		}).Draw(t, "nonString")

		compare := "test"

		result := EqualInterface(nonString, compare)

		if result {
			t.Fatalf("EqualInterface should return false for non-string interface")
		}
	})
}
