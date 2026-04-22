# Bug Report: StringDict.LoadIndex Race Condition

## Summary

`StringDict.LoadIndex()` has a **missing double-check** after acquiring the lock, causing a race condition where concurrent goroutines calling `LoadIndex()` with the same key can receive **different indices** for the same string. Additionally, there is a **data race** on the `corpusIndexes` slice, which is written to outside the lock.

## Severity

**HIGH** — This bug silently corrupts the string-to-index mapping, which can lead to:
- Wrong data lookups (string at index X is not what was stored)
- Index collisions (two strings map to the same index)
- Non-deterministic behavior in production

## Affected File

`lib/stringinterner/string_interner.go:73-91`

## Root Cause

Two issues in `LoadIndex()`:

### Issue 1: Missing double-check after lock acquisition

```go
func (s *StringDict) LoadIndex(key string) uint64 {
    vv, ok := s.corpus.Load(key)   // (1) Fast path check
    if ok {
        index, _ := vv.(uint64)
        return index
    }
    s.corpusLock.Lock()            // (2) Acquire lock
    // ⚠️ MISSING: re-check s.corpus.Load(key) here!
    s.corpusIndex = s.corpusIndex + 1
    index := s.corpusIndex
    // ... grow slice ...
    s.corpusLock.Unlock()          // (3) Release lock
    s.corpusIndexes[index] = key   // (4) DATA RACE: write to shared slice outside lock
    s.corpus.Store(key, index)     // (5) Store in map
    return index
}
```

When two goroutines both execute step (1) for the same key before either stores it:
1. Both get `ok = false`
2. Both enter the locked section (possibly sequentially)
3. Each allocates a **different index** for the same string
4. The second write to `s.corpusIndexes[index]` overwrites the first

### Issue 2: Data race on `corpusIndexes` slice

Line 88 (`s.corpusIndexes[index] = key`) writes to the shared slice **after** the mutex is unlocked on line 86. Multiple goroutines can write to this slice concurrently.

## Production Impact

`StringDict` is used in `app/ts-store/stream/tag_task.go:159`:
```go
s.stringDict = stringinterner.NewStringDict()
```

This is used for stream processing tasks where concurrent writes are expected.

## Reproduction

### With Go Race Detector

```bash
go test -v -race ./lib/stringinterner/... -run "TestStringDict_ConcurrentDeterminism"
```

### Minimal Reproduction Test

```go
package stringinterner_test

import (
    "sync"
    "testing"

    "github.com/openGemini/openGemini/lib/stringinterner"
)

func TestStringDict_RaceCondition(t *testing.T) {
    dict := stringinterner.NewStringDict()
    numGoroutines := 10
    results := make([]uint64, numGoroutines)
    var wg sync.WaitGroup

    for g := 0; g < numGoroutines; g++ {
        wg.Add(1)
        go func(gid int) {
            defer wg.Done()
            results[gid] = dict.LoadIndex("same_key")
        }(g)
    }
    wg.Wait()

    first := results[0]
    for i := 1; i < numGoroutines; i++ {
        if results[i] != first {
            t.Fatalf("Race condition: goroutine 0 got index %d, goroutine %d got index %d for key 'same_key'",
                first, i, results[i])
        }
    }
}
```

## Expected Behavior

All goroutines calling `LoadIndex("same_key")` should receive the same index value.

## Suggested Fix

Add a double-check after acquiring the lock:

```go
func (s *StringDict) LoadIndex(key string) uint64 {
    vv, ok := s.corpus.Load(key)
    if ok {
        index, _ := vv.(uint64)
        return index
    }
    s.corpusLock.Lock()
    // Double-check: another goroutine may have stored it
    if vv, ok := s.corpus.Load(key); ok {
        s.corpusLock.Unlock()
        index, _ := vv.(uint64)
        return index
    }
    s.corpusIndex = s.corpusIndex + 1
    index := s.corpusIndex
    if uint64(len(s.corpusIndexes)) <= index {
        s.corpusIndexes = append(s.corpusIndexes, EmptyStr)
        s.corpusIndexes = s.corpusIndexes[:cap(s.corpusIndexes)]
    }
    key = strings.Clone(key)
    s.corpusIndexes[index] = key
    s.corpus.Store(key, index)
    s.corpusLock.Unlock()
    return index
}
```

Note: The write to `s.corpusIndexes[index]` is also moved inside the lock.
