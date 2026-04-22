# Property-Based Testing Summary - openGemini

## Overview

**Date**: April 22, 2026
**Method**: Property-based testing using `pgregory.net/rapid` library
**Packages Tested**: 11
**Bugs Confirmed**: 4 (including 1 from previous session)
**Tests Created**: 100+ property-based tests

## Final Test Results

```
FAIL  lib/stringinterner  — Race condition in StringDict.LoadIndex
ok    lib/stream          — All tests PASS (NaN inconsistency confirmed by code analysis)
FAIL  lib/atomic          — NaN corruption + negative b bugs
ok    lib/errno           — All tests PASS
ok    lib/memory          — All tests PASS (unit mismatch confirmed by code analysis)
ok    lib/rand            — All tests PASS
ok    lib/bufferpool      — All tests PASS
ok    lib/bloomfilter     — All tests PASS (boundary bug confirmed in prior session)
ok    lib/strings         — All tests PASS
ok    lib/consistenthash  — All tests PASS
ok    lib/binarysearch    — All tests PASS
```

## Confirmed Bugs

### Bug 1: StringDict.LoadIndex Race Condition (HIGH)
- **File**: `BUG_REPORT_STRINGINTERNER.md`
- **Package**: `lib/stringinterner`
- **Test**: `TestStringDict_ConcurrentDeterminism` (confirmed with `-race`)
- **Root cause**: Missing double-check after lock + data race on shared slice
- **Impact**: Concurrent goroutines get different indices for same key
- **Production usage**: `app/ts-store/stream/tag_task.go:159`

### Bug 2: CompareAndSwapMax/MinFloat64 NaN Corruption (MEDIUM)
- **File**: `BUG_REPORT_ATOMIC_NAN.md`
- **Package**: `lib/atomic`
- **Tests**: `TestCompareAndSwapMaxFloat64_NaNInput`, `TestCompareAndSwapMinFloat64_NaNInput`
- **Root cause**: `math.Max(u, NaN)` returns NaN; NaN != u triggers CAS swap
- **Impact**: Single NaN value permanently corrupts aggregate state
- **Production usage**: `lib/stream/stream.go:72,74` (min/max aggregation)

### Bug 3: Memory defaultMaxMem Unit Mismatch (MEDIUM)
- **File**: `BUG_REPORT_MEMORY_UNIT_MISMATCH.md`
- **Package**: `lib/memory`
- **Test**: Code analysis (can't trigger without mocking gopsutil)
- **Root cause**: `64 << 30` is bytes, but API returns kB → 1024x overestimate
- **Impact**: Wrong cache/memory sizing on gopsutil failure paths
- **Production usage**: `lib/config/store.go:384`, `lib/config/readcache.go:40`, `engine/immutable/hot.go:184`

### Bug 4: BloomFilter Size Validation (MEDIUM) [Previous Session]
- **File**: `BUG_REPORT_BLOOMFILTER.md`
- **Package**: `lib/bloomfilter`
- **Test**: `bloomfilter_boundary_test.go`
- **Root cause**: API accepts any size but only specific sizes work correctly
- **Impact**: Panics on invalid sizes

## Dismissed (NOT Bugs)

| Finding | Package | Reason |
|---------|---------|--------|
| BufferPool capacity preservation | lib/bufferpool | Expected behavior (proven by existing test `TestBufferPool`) |
| SetModInt64AndADD negative b | lib/atomic | Latent design flaw; production only uses b=±1 which works |
| Rand Int63 upper bound | lib/rand | False alarm — test had wrong expected range |

## Test Files Created

| Package | File | Tests |
|---------|------|-------|
| lib/stringinterner | string_interner_rapid_test.go | 11 |
| lib/stream | stream_rapid_test.go | 14 |
| lib/atomic | atomic_rapid_test.go | 14 |
| lib/errno | errno_rapid_test.go | 15 |
| lib/memory | memory_rapid_test.go | 5 |
| lib/rand | rand_rapid_test.go | 10 |
| lib/bufferpool | bufferpool_rapid_test.go | 10 |
| lib/bufferpool | behavior_verification_test.go | 10 |
| lib/bloomfilter | bloomfilter_rapid_test.go | 11 |
| lib/bloomfilter | bloomfilter_boundary_test.go | 8 |
| lib/strings | strings_rapid_test.go | 17 |
| lib/consistenthash | consistenthash_rapid_test.go | 8 |
| lib/binarysearch | binarysearch_rapid_test.go | 9 |

## Bug Report Files

| File | Bug | Severity |
|------|-----|----------|
| BUG_REPORT_STRINGINTERNER.md | StringDict race condition | HIGH |
| BUG_REPORT_ATOMIC_NAN.md | NaN value corruption | MEDIUM |
| BUG_REPORT_MEMORY_UNIT_MISMATCH.md | kB vs bytes mismatch | MEDIUM |
| BUG_REPORT_BLOOMFILTER.md | Missing size validation | MEDIUM |

## How to Run

```bash
# Run all PBT tests
go test ./lib/stringinterner/... ./lib/stream/... ./lib/atomic/... \
       ./lib/errno/... ./lib/memory/... ./lib/rand/... \
       ./lib/bufferpool/... ./lib/bloomfilter/... ./lib/strings/... \
       ./lib/consistenthash/... ./lib/binarysearch/...

# Reproduce specific bugs
go test -race ./lib/stringinterner/... -run "TestStringDict_ConcurrentDeterminism"
go test ./lib/atomic/... -run "NaNInput"
go test ./lib/bloomfilter/... -run "BoundaryPanic"
```
