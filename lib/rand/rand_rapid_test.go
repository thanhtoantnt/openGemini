package rand_test

import (
	"math"
	"testing"

	"github.com/openGemini/openGemini/lib/rand"
	"pgregory.net/rapid"
)

func TestInt63_NonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rand.Int63()
		if v < 0 {
			t.Fatalf("Int63() = %d; want >= 0", v)
		}
	})
}

func TestIntn_Range(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, math.MaxInt32).Draw(t, "n")
		v := rand.Intn(n)
		if v < 0 || v >= n {
			t.Fatalf("Intn(%d) = %d; want [0, %d)", n, v, n)
		}
	})
}

func TestInt63n_Range(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.Int64Range(1, math.MaxInt64>>1).Draw(t, "n")
		v := rand.Int63n(n)
		if v < 0 || v >= n {
			t.Fatalf("Int63n(%d) = %d; want [0, %d)", n, v, n)
		}
	})
}

func TestInt31n_Range(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.Int32Range(1, math.MaxInt32).Draw(t, "n")
		v := rand.Int31n(n)
		if v < 0 || v >= n {
			t.Fatalf("Int31n(%d) = %d; want [0, %d)", n, v, n)
		}
	})
}

func TestFloat64_Range(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rand.Float64()
		if v < 0.0 || v >= 1.0 {
			t.Fatalf("Float64() = %v; want [0, 1)", v)
		}
	})
}

func TestIntn_One(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rand.Intn(1)
		if v != 0 {
			t.Fatalf("Intn(1) = %d; want 0", v)
		}
	})
}

func TestInt63n_One(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rand.Int63n(1)
		if v != 0 {
			t.Fatalf("Int63n(1) = %d; want 0", v)
		}
	})
}

func TestInt31n_One(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rand.Int31n(1)
		if v != 0 {
			t.Fatalf("Int31n(1) = %d; want 0", v)
		}
	})
}

func TestIntn_EdgeCases(t *testing.T) {
	for _, n := range []int{1, 2, 3, 10, 100, 1000} {
		for i := 0; i < 100; i++ {
			v := rand.Intn(n)
			if v < 0 || v >= n {
				t.Fatalf("Intn(%d) = %d; out of range", n, v)
			}
		}
	}
}
