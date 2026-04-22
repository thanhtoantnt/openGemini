package atomic_test

import (
	"math"
	"testing"

	"github.com/openGemini/openGemini/lib/atomic"
	"pgregory.net/rapid"
)

func TestCompareAndSwapMaxInt64_Monotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := rapid.Int64().Draw(t, "initial")
		b := rapid.Int64().Draw(t, "b")

		var a = initial
		result := atomic.CompareAndSwapMaxInt64(&a, b)

		if a < initial {
			t.Fatalf("Max decreased value: %d -> %d (b=%d)", initial, a, b)
		}
		if a < b {
			t.Fatalf("Max result %d < b %d", a, b)
		}
		_ = result
	})
}

func TestCompareAndSwapMinInt64_AntiMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := rapid.Int64().Draw(t, "initial")
		b := rapid.Int64().Draw(t, "b")

		var a = initial
		result := atomic.CompareAndSwapMinInt64(&a, b)

		if a > initial {
			t.Fatalf("Min increased value: %d -> %d (b=%d)", initial, a, b)
		}
		if a > b {
			t.Fatalf("Min result %d > b %d", a, b)
		}
		_ = result
	})
}

func TestCompareAndSwapMaxInt64_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int64().Draw(t, "v")
		var a = v
		atomic.CompareAndSwapMaxInt64(&a, v)
		if a != v {
			t.Fatalf("Max(a, a) changed value from %d to %d", v, a)
		}
	})
}

func TestCompareAndSwapMinInt64_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int64().Draw(t, "v")
		var a = v
		atomic.CompareAndSwapMinInt64(&a, v)
		if a != v {
			t.Fatalf("Min(a, a) changed value from %d to %d", v, a)
		}
	})
}

func TestCompareAndSwapMaxFloat64_Monotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := rapid.Float64().Draw(t, "initial")
		b := rapid.Float64().Draw(t, "b")

		var a = initial
		atomic.CompareAndSwapMaxFloat64(&a, b)

		if a < initial {
			t.Fatalf("Max decreased value: %v -> %v (b=%v)", initial, a, b)
		}
	})
}

func TestCompareAndSwapMinFloat64_AntiMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initial := rapid.Float64().Draw(t, "initial")
		b := rapid.Float64().Draw(t, "b")

		var a = initial
		atomic.CompareAndSwapMinFloat64(&a, b)

		if a > initial {
			t.Fatalf("Min increased value: %v -> %v (b=%v)", initial, a, b)
		}
	})
}

func TestAddFloat64_Identity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Float64().Draw(t, "v")
		var a = v
		result := atomic.AddFloat64(&a, 0.0)
		if result != v || a != v {
			t.Fatalf("AddFloat64(%v, 0) = %v, a=%v", v, result, a)
		}
	})
}

func TestAddFloat64_Inverse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Float64Range(-1e15, 1e15).Draw(t, "v")
		var a = v
		atomic.AddFloat64(&a, -v)
		if a != 0.0 {
			t.Fatalf("AddFloat64(%v, %v) = %v; want 0", v, -v, a)
		}
	})
}

func TestSetModInt64AndADD_NonNegRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a0 := rapid.Int64Range(0, 1000).Draw(t, "a")
		b := rapid.Int64Range(0, 1000).Draw(t, "b")
		mod := rapid.Int64Range(1, 1000).Draw(t, "mod")

		var a = a0
		result := atomic.SetModInt64AndADD(&a, b, mod)

		if result < 0 || result >= mod {
			t.Fatalf("SetMod(%d, %d, %d) = %d; want [0, %d)", a0, b, mod, result, mod)
		}
	})
}

func TestSetModInt64AndADD_MatchesPlainMath(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a0 := rapid.Int64Range(0, 1000).Draw(t, "a")
		b := rapid.Int64Range(0, 1000).Draw(t, "b")
		mod := rapid.Int64Range(1, 1000).Draw(t, "mod")

		var a = a0
		result := atomic.SetModInt64AndADD(&a, b, mod)

		expected := (a0 + b) % mod
		if result != expected {
			t.Fatalf("SetMod(%d, %d, %d) = %d; want %d", a0, b, mod, result, expected)
		}
	})
}

func TestLoadModInt64AndADD_DoesNotModify(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a0 := rapid.Int64Range(0, 1000).Draw(t, "a")
		b := rapid.Int64Range(0, 1000).Draw(t, "b")
		mod := rapid.Int64Range(1, 1000).Draw(t, "mod")

		var a = a0
		result := atomic.LoadModInt64AndADD(&a, b, mod)

		if a != a0 {
			t.Fatalf("LoadMod modified *a from %d to %d", a0, a)
		}

		expected := (a0 + b) % mod
		if result != expected {
			t.Fatalf("LoadMod(%d, %d, %d) = %d; want %d", a0, b, mod, result, expected)
		}
	})
}

func TestSetModInt64AndADD_NegativeB(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a0 := rapid.Int64Range(0, 100).Draw(t, "a")
		b := rapid.Int64Range(-100, -1).Draw(t, "b")
		mod := rapid.Int64Range(1, 100).Draw(t, "mod")

		var a = a0
		result := atomic.SetModInt64AndADD(&a, b, mod)

		if result < 0 || result >= mod {
			t.Fatalf("SetMod(%d, %d, %d) = %d; want [0, %d) — negative b not handled", a0, b, mod, result, mod)
		}
	})
}

func TestCompareAndSwapMaxFloat64_NaNInput(t *testing.T) {
	var a float64 = 5.0
	atomic.CompareAndSwapMaxFloat64(&a, math.NaN())
	if math.IsNaN(a) {
		t.Fatalf("CompareAndSwapMaxFloat64(5.0, NaN) corrupted value to NaN")
	}
}

func TestCompareAndSwapMinFloat64_NaNInput(t *testing.T) {
	var a float64 = 5.0
	atomic.CompareAndSwapMinFloat64(&a, math.NaN())
	if math.IsNaN(a) {
		t.Fatalf("CompareAndSwapMinFloat64(5.0, NaN) corrupted value to NaN")
	}
}

func TestDoubleMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a0 := rapid.Int64().Draw(t, "a0")
		x := rapid.Int64().Draw(t, "x")
		y := rapid.Int64().Draw(t, "y")

		var a = a0
		atomic.CompareAndSwapMaxInt64(&a, x)
		atomic.CompareAndSwapMaxInt64(&a, y)

		var b = a0
		atomic.CompareAndSwapMaxInt64(&b, maxOf(x, y))

		if a != b {
			t.Fatalf("Max(Max(%d,%d),%d)=%d; Max(%d,%d)=%d", a0, x, y, a, a0, maxOf(x, y), b)
		}
	})
}

func maxOf(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
