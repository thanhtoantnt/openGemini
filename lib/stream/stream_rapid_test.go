package stream_test

import (
	"math"
	"sort"
	"testing"

	"github.com/openGemini/openGemini/lib/stream"
	"pgregory.net/rapid"
)

func TestSingleThreadMin_Commutativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "min", false)
		r1 := fc.SingleThreadFunc(a, b)
		r2 := fc.SingleThreadFunc(b, a)
		if r1 != r2 {
			t.Fatalf("min not commutative: min(%v,%v)=%v, min(%v,%v)=%v", a, b, r1, b, a, r2)
		}
	})
}

func TestSingleThreadMax_Commutativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "max", false)
		r1 := fc.SingleThreadFunc(a, b)
		r2 := fc.SingleThreadFunc(b, a)
		if r1 != r2 {
			t.Fatalf("max not commutative: max(%v,%v)=%v, max(%v,%v)=%v", a, b, r1, b, a, r2)
		}
	})
}

func TestSingleThreadMin_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "min", false)
		r := fc.SingleThreadFunc(a, a)
		if r != a {
			t.Fatalf("min(a,a)=%v; want %v", r, a)
		}
	})
}

func TestSingleThreadMax_Idempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "max", false)
		r := fc.SingleThreadFunc(a, a)
		if r != a {
			t.Fatalf("max(a,a)=%v; want %v", r, a)
		}
	})
}

func TestSingleThreadSum_MatchesFold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "n")
		vals := make([]float64, n)
		for i := range vals {
			vals[i] = rapid.Float64Range(-1000, 1000).Draw(t, "val")
		}

		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "sum", false)
		result := 0.0
		for _, v := range vals {
			result = fc.SingleThreadFunc(result, v)
		}

		expected := 0.0
		for _, v := range vals {
			expected += v
		}
		if result != expected {
			t.Fatalf("sum fold = %v; want %v", result, expected)
		}
	})
}

func TestSingleThreadMin_MatchesFold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "n")
		vals := make([]float64, n)
		for i := range vals {
			vals[i] = rapid.Float64Range(-1000, 1000).Draw(t, "val")
		}

		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "min", false)
		result := vals[0]
		for _, v := range vals[1:] {
			result = fc.SingleThreadFunc(result, v)
		}

		sort.Float64s(vals)
		expected := vals[0]
		if result != expected {
			t.Fatalf("min fold = %v; want %v (sorted vals[0])", result, expected)
		}
	})
}

func TestSingleThreadMax_MatchesFold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "n")
		vals := make([]float64, n)
		for i := range vals {
			vals[i] = rapid.Float64Range(-1000, 1000).Draw(t, "val")
		}

		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "max", false)
		result := vals[0]
		for _, v := range vals[1:] {
			result = fc.SingleThreadFunc(result, v)
		}

		sort.Float64s(vals)
		expected := vals[n-1]
		if result != expected {
			t.Fatalf("max fold = %v; want %v (sorted vals[-1])", result, expected)
		}
	})
}

func TestSingleThreadMin_MatchesMathMin(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "min", false)
		r := fc.SingleThreadFunc(a, b)
		if r != math.Min(a, b) {
			t.Fatalf("singleThreadMin(%v,%v)=%v; math.Min=%v", a, b, r, math.Min(a, b))
		}
	})
}

func TestSingleThreadMax_MatchesMathMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "max", false)
		r := fc.SingleThreadFunc(a, b)
		if r != math.Max(a, b) {
			t.Fatalf("singleThreadMax(%v,%v)=%v; math.Max=%v", a, b, r, math.Max(a, b))
		}
	})
}

func TestConcurrencyFunc_SumFold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "n")
		vals := make([]float64, n)
		for i := range vals {
			vals[i] = rapid.Float64Range(-1000, 1000).Draw(t, "val")
		}

		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "sum", true)
		result := 0.0
		for _, v := range vals {
			result = fc.ConcurrencyFunc(&result, v)
		}

		expected := 0.0
		for _, v := range vals {
			expected += v
		}
		if result != expected {
			t.Fatalf("concurrent sum fold = %v; want %v", result, expected)
		}
	})
}

func TestConcurrencyFunc_MinMatchesMathMin(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "min", true)
		val := a
		r := fc.ConcurrencyFunc(&val, b)
		if r != math.Min(a, b) {
			t.Fatalf("concurrentMin(%v,%v)=%v; math.Min=%v", a, b, r, math.Min(a, b))
		}
	})
}

func TestConcurrencyFunc_MaxMatchesMathMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Float64().Draw(t, "a")
		b := rapid.Float64().Draw(t, "b")
		fc, _ := stream.NewFieldCall(0, 0, "test", "test", "max", true)
		val := a
		r := fc.ConcurrencyFunc(&val, b)
		if r != math.Max(a, b) {
			t.Fatalf("concurrentMax(%v,%v)=%v; math.Max=%v", a, b, r, math.Max(a, b))
		}
	})
}

func TestFieldCalls_Sort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 20).Draw(t, "n")
		fcs := make(stream.FieldCalls, n)
		for i := range fcs {
			fcs[i] = &stream.FieldCall{Alias: rapid.String().Draw(t, "alias")}
		}
		sort.Sort(fcs)
		for i := 1; i < len(fcs); i++ {
			if fcs[i-1].Alias > fcs[i].Alias {
				t.Fatalf("not sorted at index %d: %q > %q", i-1, fcs[i-1].Alias, fcs[i].Alias)
			}
		}
	})
}

func TestNewFieldCall_UnknownCall(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		call := rapid.String().Draw(t, "call")
		if call == "min" || call == "max" || call == "sum" || call == "count" {
			return
		}
		_, err := stream.NewFieldCall(0, 0, "test", "test", call, false)
		if err == nil {
			t.Fatalf("expected error for unknown call %q", call)
		}
	})
}

func TestNewFieldCall_KnownCalls(t *testing.T) {
	for _, call := range []string{"min", "max", "sum", "count"} {
		fc, err := stream.NewFieldCall(0, 0, "test", "test", call, false)
		if err != nil {
			t.Fatalf("NewFieldCall with call=%q returned error: %v", call, err)
		}
		if fc.SingleThreadFunc == nil {
			t.Fatalf("NewFieldCall with call=%q: SingleThreadFunc is nil", call)
		}
	}
	for _, call := range []string{"min", "max", "sum", "count"} {
		fc, err := stream.NewFieldCall(0, 0, "test", "test", call, true)
		if err != nil {
			t.Fatalf("NewFieldCall with call=%q returned error: %v", call, err)
		}
		if fc.ConcurrencyFunc == nil {
			t.Fatalf("NewFieldCall with call=%q: ConcurrencyFunc is nil", call)
		}
	}
}
