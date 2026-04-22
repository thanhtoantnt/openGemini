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

package binarysearch_test

import (
	"sort"
	"testing"

	"pgregory.net/rapid"

	"github.com/openGemini/openGemini/lib/binarysearch"
)

// TestUpperBoundInt64Ascending_EmptyArray
// Verifies behavior with empty array
func TestUpperBoundInt64Ascending_EmptyArray(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var a []int64
		x := rapid.Int64().Draw(t, "x")

		result := binarysearch.UpperBoundInt64Ascending(a, x)

		if result != -1 {
			t.Fatalf("UpperBound on empty array should return -1, got %d", result)
		}
	})
}

// TestUpperBoundInt64Ascending_SingleElement
// Verifies behavior with single element
func TestUpperBoundInt64Ascending_SingleElement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elem := rapid.Int64().Draw(t, "elem")
		x := rapid.Int64().Draw(t, "x")

		a := []int64{elem}
		result := binarysearch.UpperBoundInt64Ascending(a, x)

		if x <= elem {
			if result != 0 {
				t.Fatalf("UpperBound for x=%d <= elem=%d should return 0, got %d", x, elem, result)
			}
		} else {
			if result != -1 {
				t.Fatalf("UpperBound for x=%d > elem=%d should return -1, got %d", x, elem, result)
			}
		}
	})
}

// TestUpperBoundInt64Ascending_SortedArray
// Verifies that result is correct for sorted arrays
func TestUpperBoundInt64Ascending_SortedArray(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")

		a := make([]int64, n)
		for i := 0; i < n; i++ {
			a[i] = rapid.Int64Range(-1000, 1000).Draw(t, "elem")
		}
		sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })

		x := rapid.Int64Range(-1000, 1000).Draw(t, "x")

		result := binarysearch.UpperBoundInt64Ascending(a, x)

		if result == -1 {
			// All elements should be < x
			if a[len(a)-1] >= x {
				t.Fatalf("UpperBound returned -1 but a[%d]=%d >= x=%d", len(a)-1, a[len(a)-1], x)
			}
		} else {
			if result < 0 || result >= len(a) {
				t.Fatalf("UpperBound returned invalid index %d for array of length %d", result, len(a))
			}

			// a[result] should be >= x
			if a[result] < x {
				t.Fatalf("UpperBound returned index %d but a[%d]=%d < x=%d", result, result, a[result], x)
			}

			// This should be the first such element
			if result > 0 && a[result-1] >= x {
				t.Fatalf("UpperBound returned %d but a[%d]=%d >= x=%d (not first)", result, result-1, a[result-1], x)
			}
		}
	})
}

// TestLowerBoundInt64Ascending_SortedArray
// Verifies that result is correct for sorted arrays
func TestLowerBoundInt64Ascending_SortedArray(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")

		a := make([]int64, n)
		for i := 0; i < n; i++ {
			a[i] = rapid.Int64Range(-1000, 1000).Draw(t, "elem")
		}
		sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })

		x := rapid.Int64Range(-1000, 1000).Draw(t, "x")

		result := binarysearch.LowerBoundInt64Ascending(a, x)

		if result == -1 {
			// First element should be >= x
			if a[0] < x {
				t.Fatalf("LowerBound returned -1 but a[0]=%d < x=%d", a[0], x)
			}
		} else {
			if result < 0 || result >= len(a) {
				t.Fatalf("LowerBound returned invalid index %d for array of length %d", result, len(a))
			}

			// a[result] should be < x
			if a[result] >= x {
				t.Fatalf("LowerBound returned index %d but a[%d]=%d >= x=%d", result, result, a[result], x)
			}

			// This should be the last such element
			if result < len(a)-1 && a[result+1] < x {
				t.Fatalf("LowerBound returned %d but a[%d]=%d < x=%d (not last)", result, result+1, a[result+1], x)
			}
		}
	})
}

// TestUpperBoundInt64Descending_SortedArray
// Verifies that result is correct for descending sorted arrays
func TestUpperBoundInt64Descending_SortedArray(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")

		a := make([]int64, n)
		for i := 0; i < n; i++ {
			a[i] = rapid.Int64Range(-1000, 1000).Draw(t, "elem")
		}
		sort.Slice(a, func(i, j int) bool { return a[i] > a[j] })

		x := rapid.Int64Range(-1000, 1000).Draw(t, "x")

		result := binarysearch.UpperBoundInt64Descending(a, x)

		if result == -1 {
			// First element should be < x
			if a[0] >= x {
				t.Fatalf("UpperBound descending returned -1 but a[0]=%d >= x=%d", a[0], x)
			}
		} else {
			if result < 0 || result >= len(a) {
				t.Fatalf("UpperBound descending returned invalid index %d for array of length %d", result, len(a))
			}

			// a[result] should be >= x
			if a[result] < x {
				t.Fatalf("UpperBound descending returned index %d but a[%d]=%d < x=%d", result, result, a[result], x)
			}
		}
	})
}

// TestLowerBoundInt64Descending_SortedArray
// Verifies that result is correct for descending sorted arrays
func TestLowerBoundInt64Descending_SortedArray(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")

		a := make([]int64, n)
		for i := 0; i < n; i++ {
			a[i] = rapid.Int64Range(-1000, 1000).Draw(t, "elem")
		}
		sort.Slice(a, func(i, j int) bool { return a[i] > a[j] })

		x := rapid.Int64Range(-1000, 1000).Draw(t, "x")

		result := binarysearch.LowerBoundInt64Descending(a, x)

		if result == -1 {
			// Last element should be >= x
			if a[len(a)-1] < x {
				t.Fatalf("LowerBound descending returned -1 but a[%d]=%d < x=%d", len(a)-1, a[len(a)-1], x)
			}
		} else {
			if result < 0 || result >= len(a) {
				t.Fatalf("LowerBound descending returned invalid index %d for array of length %d", result, len(a))
			}

			// a[result] should be < x
			if a[result] >= x {
				t.Fatalf("LowerBound descending returned index %d but a[%d]=%d >= x=%d", result, result, a[result], x)
			}
		}
	})
}

// TestUpperBoundInt64Ascending_AllSame
// Verifies behavior when all elements are the same
func TestUpperBoundInt64Ascending_AllSame(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")
		elem := rapid.Int64().Draw(t, "elem")
		x := rapid.Int64().Draw(t, "x")

		a := make([]int64, n)
		for i := range a {
			a[i] = elem
		}

		result := binarysearch.UpperBoundInt64Ascending(a, x)

		if x <= elem {
			if result != 0 {
				t.Fatalf("UpperBound for x=%d <= elem=%d should return 0, got %d", x, elem, result)
			}
		} else {
			if result != -1 {
				t.Fatalf("UpperBound for x=%d > elem=%d should return -1, got %d", x, elem, result)
			}
		}
	})
}

// TestUpperBoundInt64Ascending_BeyondRange
// Verifies behavior when x is beyond array range
func TestUpperBoundInt64Ascending_BeyondRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 100).Draw(t, "n")
		minVal := rapid.Int64Range(-1000, 1000).Draw(t, "minVal")
		maxVal := rapid.Int64Range(minVal, minVal+1000).Draw(t, "maxVal")

		a := make([]int64, n)
		for i := 0; i < n; i++ {
			a[i] = minVal + int64(i)*(maxVal-minVal)/int64(n)
		}
		sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })

		// Test with value beyond max
		x := maxVal + 1000
		result := binarysearch.UpperBoundInt64Ascending(a, x)
		if result != -1 {
			t.Fatalf("UpperBound for x=%d > max=%d should return -1, got %d", x, a[len(a)-1], result)
		}

		// Test with value below min
		x = minVal - 1000
		result = binarysearch.UpperBoundInt64Ascending(a, x)
		if result != 0 {
			t.Fatalf("UpperBound for x=%d < min=%d should return 0, got %d", x, a[0], result)
		}
	})
}
