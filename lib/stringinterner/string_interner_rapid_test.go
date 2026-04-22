package stringinterner_test

import (
	"sync"
	"testing"

	"github.com/openGemini/openGemini/lib/stringinterner"
	"pgregory.net/rapid"
)

func TestStringDict_LoadIndexRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		s := rapid.String().Draw(t, "s")

		idx := dict.LoadIndex(s)
		got := dict.LoadValue(int(idx))

		if got != s {
			t.Fatalf("LoadIndex(%q)=%d, LoadValue(%d)=%q; want %q", s, idx, idx, got, s)
		}
	})
}

func TestStringDict_LoadIndexDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		s := rapid.String().Draw(t, "s")

		idx1 := dict.LoadIndex(s)
		idx2 := dict.LoadIndex(s)

		if idx1 != idx2 {
			t.Fatalf("LoadIndex(%q) returned different indices: %d vs %d", s, idx1, idx2)
		}
	})
}

func TestStringDict_DifferentStringsDifferentIndices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		s1 := rapid.String().Draw(t, "s1")
		s2 := rapid.String().Draw(t, "s2")
		if s1 == s2 {
			return
		}

		idx1 := dict.LoadIndex(s1)
		idx2 := dict.LoadIndex(s2)

		if idx1 == idx2 {
			t.Fatalf("Different strings %q and %q got same index %d", s1, s2, idx1)
		}
	})
}

func TestStringDict_LoadValueZeroIsEmpty(t *testing.T) {
	dict := stringinterner.NewStringDict()
	got := dict.LoadValue(0)
	if got != "" {
		t.Fatalf("LoadValue(0) = %q; want empty string", got)
	}
}

func TestStringDict_ManyStringsRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		n := rapid.IntRange(1, 50).Draw(t, "n")

		strings := make([]string, n)
		for i := 0; i < n; i++ {
			strings[i] = rapid.String().Draw(t, "s")
		}

		for _, s := range strings {
			dict.LoadIndex(s)
		}

		for _, s := range strings {
			idx := dict.LoadIndex(s)
			got := dict.LoadValue(int(idx))
			if got != s {
				t.Fatalf("Roundtrip failed for %q: LoadValue(%d)=%q", s, idx, got)
			}
		}
	})
}

func TestStringDict_ConcurrentDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		n := rapid.IntRange(1, 20).Draw(t, "n")
		numGoroutines := rapid.IntRange(2, 8).Draw(t, "goroutines")

		results := make([][]uint64, numGoroutines)
		var wg sync.WaitGroup

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(gid int) {
				defer wg.Done()
				indices := make([]uint64, n)
				for i := 0; i < n; i++ {
					indices[i] = dict.LoadIndex("key")
				}
				results[gid] = indices
			}(g)
		}
		wg.Wait()

		for g := 1; g < numGoroutines; g++ {
			for i := 0; i < n; i++ {
				if results[0][i] != results[g][i] {
					t.Fatalf("Concurrent LoadIndex returned different values: goroutine 0 got %d, goroutine %d got %d at call %d",
						results[0][i], g, results[g][i], i)
				}
			}
		}
	})
}

func TestStringDict_ConcurrentRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := stringinterner.NewStringDict()
		keys := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
		numGoroutines := rapid.IntRange(2, 8).Draw(t, "goroutines")
		iters := rapid.IntRange(10, 100).Draw(t, "iters")

		var wg sync.WaitGroup
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(gid int) {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					k := keys[i%len(keys)]
					idx := dict.LoadIndex(k)
					got := dict.LoadValue(int(idx))
					if got != k {
						t.Errorf("goroutine %d: LoadValue(LoadIndex(%q)) = %q", gid, k, got)
					}
				}
			}(g)
		}
		wg.Wait()
	})
}

func TestInternSafe_Identity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		got := stringinterner.InternSafe(s)
		if got != s {
			t.Fatalf("InternSafe(%q) = %q; want %q", s, got, s)
		}
	})
}

func TestInternSafe_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		r1 := stringinterner.InternSafe(s)
		r2 := stringinterner.InternSafe(s)
		if r1 != r2 {
			t.Fatalf("InternSafe not idempotent: %q vs %q", r1, r2)
		}
	})
}

func TestInternTagValue_Identity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		got := stringinterner.InternTagValue(s)
		if got != s {
			t.Fatalf("InternTagValue(%q) = %q; want %q", s, got, s)
		}
	})
}

func TestInternSafe_Concurrent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numGoroutines := rapid.IntRange(2, 8).Draw(t, "goroutines")
		iters := rapid.IntRange(10, 100).Draw(t, "iters")

		var wg sync.WaitGroup
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					s := "shared_string"
					got := stringinterner.InternSafe(s)
					if got != s {
						t.Errorf("InternSafe(%q) = %q", s, got)
					}
				}
			}()
		}
		wg.Wait()
	})
}
