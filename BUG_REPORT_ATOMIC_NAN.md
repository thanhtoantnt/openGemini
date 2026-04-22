# Bug Report: CompareAndSwapMax/MinFloat64 NaN Value Corruption

## Summary

`CompareAndSwapMaxFloat64()` and `CompareAndSwapMinFloat64()` silently corrupt stored values when `NaN` is passed as the argument. Instead of being a no-op (since NaN is neither greater nor less than any value), the functions replace the stored value with NaN.

## Severity

**MEDIUM** — In a time series database, NaN values can appear in float data (from division by zero, invalid measurements, etc.). When used in stream aggregation (min/max), a single NaN value will permanently corrupt the aggregate state.

## Affected Files

- `lib/atomic/float64.go:42-54` (`CompareAndSwapMaxFloat64`)
- `lib/atomic/float64.go:56-68` (`CompareAndSwapMinFloat64`)
- `lib/stream/stream.go:72,74` (consumer: min/max aggregation)

## Root Cause

The functions use `math.Max(u, b) == u` / `math.Min(u, b) == u` to check whether an update is needed. In Go 1.21+, `math.Max(x, NaN)` returns `NaN`, and `NaN == anything` is always `false`.

```go
func CompareAndSwapMaxFloat64(a *float64, b float64) float64 {
    p := (*uint64)(unsafe.Pointer(a))
    for {
        v := atomic.LoadUint64(p)
        u := math.Float64frombits(v)   // u = 5.0
        if math.Max(u, b) == u {        // math.Max(5.0, NaN) = NaN; NaN == 5.0 → false
            return u                    // NOT taken
        }
        // CAS replaces 5.0 with NaN!
        if atomic.CompareAndSwapUint64(p, v, math.Float64bits(b)) {
            return b                    // Returns NaN
        }
    }
}
```

## Production Impact

These functions are used as the `ConcurrencyFunc` for min/max aggregation in stream processing:

```go
// lib/stream/stream.go:72-74
case "min":
    fieldCall.ConcurrencyFunc = atomic2.CompareAndSwapMinFloat64
case "max":
    fieldCall.ConcurrencyFunc = atomic2.CompareAndSwapMaxFloat64
```

Called in `app/ts-store/stream/tag_task.go:613,691,762`:
```go
s.fieldCalls[c].ConcurrencyFunc(vs[id], curVal)
```

If `curVal` is NaN (from a measurement error or computation), the aggregate state `vs[id]` is permanently corrupted to NaN, and all subsequent aggregations will produce NaN.

## Reproduction

```go
package atomic_test

import (
    "math"
    "testing"

    "github.com/openGemini/openGemini/lib/atomic"
)

func TestCompareAndSwapMaxFloat64_NaNCorruption(t *testing.T) {
    var a float64 = 5.0
    atomic.CompareAndSwapMaxFloat64(&a, math.NaN())
    if math.IsNaN(a) {
        t.Fatalf("CompareAndSwapMaxFloat64(5.0, NaN) corrupted value to NaN; expected 5.0")
    }
}

func TestCompareAndSwapMinFloat64_NaNCorruption(t *testing.T) {
    var a float64 = 5.0
    atomic.CompareAndSwapMinFloat64(&a, math.NaN())
    if math.IsNaN(a) {
        t.Fatalf("CompareAndSwapMinFloat64(5.0, NaN) corrupted value to NaN; expected 5.0")
    }
}
```

### Run:
```bash
go test -v ./lib/atomic/... -run "NaNCorruption"
# --- FAIL: TestCompareAndSwapMaxFloat64_NaNCorruption
# --- FAIL: TestCompareAndSwapMinFloat64_NaNCorruption
```

## Expected Behavior

NaN should be treated as a no-op for max/min operations:
- `CompareAndSwapMaxFloat64(&a, NaN)` should leave `a` unchanged
- `CompareAndSwapMinFloat64(&a, NaN)` should leave `a` unchanged

This matches the behavior of `math.Max(x, NaN)` in Go <1.21 (which returned `x`) and is the safer behavior for aggregation.

## Suggested Fix

Add an explicit NaN check before the comparison:

```go
func CompareAndSwapMaxFloat64(a *float64, b float64) float64 {
    if math.IsNaN(b) {
        return atomic.LoadFloat64(a) // no-op for NaN
    }
    p := (*uint64)(unsafe.Pointer(a))
    for {
        v := atomic.LoadUint64(p)
        u := math.Float64frombits(v)
        if u >= b {
            return u
        }
        if atomic.CompareAndSwapUint64(p, v, math.Float64bits(b)) {
            return b
        }
    }
}
```

Also consider handling NaN in the stored value (`u`) to prevent the infinite loop case with non-canonical NaN bit patterns.
