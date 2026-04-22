# Bug Report: Memory defaultMaxMem Unit Mismatch (bytes vs kB)

## Summary

`defaultMaxMem` is defined as `64 << 30` (64 GiB in **bytes**), but `SysMem()` is documented to return values in **kB**. On failure fallback paths, this causes consumers to compute a memory size of ~64 TB instead of the intended 64 GB.

## Severity

**MEDIUM** — Only triggers when the system memory reader (`gopsutil`) fails (e.g., in containers, restricted environments, or edge-case systems). When triggered, it causes a 1024x overestimate of system memory, leading to incorrect cache sizing and memory limits.

## Affected Files

- `lib/memory/sysmemory_monitor.go:20` — `defaultMaxMem = 64 << 30`
- `lib/memory/sysmemory.go:98` — `return defaultMaxMem, 0`
- `lib/memory/sysmemory.go:78` — `total, available = defaultMaxMem, defaultMaxMem`
- `lib/config/store.go:384-385` — `size, _ := memory.GetMemMonitor().SysMem(); memorySize = toml.Size(size * KB)`
- `lib/config/readcache.go:40-41` — same pattern
- `engine/immutable/hot.go:184-185` — `config.KB * memoryTotal * ...`

## Root Cause

The API contract (from comments) states that `SysMem()` returns values in **kB**:

```go
// get the total amount of memory and remaining capacity (total, available (kB))
func (m *memoryMonitor) SysMem() (total, available int64) {
```

The normal path correctly converts to kB:
```go
func ReadSysMemory() (int64, int64) {
    if info, err := mem.VirtualMemory(); err != nil {
        return defaultMaxMem, 0  // ⚠️ Returns bytes, not kB!
    } else {
        return int64(info.Total >> 10), int64(info.Available >> 10)  // Correctly converts to kB
    }
}
```

But `defaultMaxMem = 64 << 30 = 68,719,476,736` is 64 GiB in **bytes**, not kB.

All consumers multiply by `KB = 1024`:
```go
// lib/config/store.go:384-385
size, _ := memory.GetMemMonitor().SysMem()
memorySize = toml.Size(size * KB)
```

### On success path:
- `SysMem()` returns e.g. `16777216` (16 GB in kB)
- Consumer: `16777216 * 1024 = 17,179,869,184` bytes (16 GB) ✓

### On failure path:
- `SysMem()` returns `defaultMaxMem = 68719476736` (64 GiB in bytes, not kB!)
- Consumer: `68719476736 * 1024 = 70,368,744,177,664` bytes ≈ **64 TB** ✗

## Expected Behavior

`defaultMaxMem` should be `64 << 20` (67,108,864 kB = 64 GiB) to match the kB API contract.

## Why This Is Hard to Test

The bug only triggers when `mem.VirtualMemory()` returns an error. On real systems with working system info, the normal path works correctly. The bug manifests in:
- Containers without `/proc/meminfo` or `/sys/fs/cgroup/memory.*`
- macOS sandboxed applications
- CI environments with restricted system access
- Edge-case virtualization platforms

## Verification (Code Analysis)

```go
// defaultMaxMem in kB would be:
// 64 GB = 64 * 1024 * 1024 kB = 67,108,864 kB
// 64 << 20 = 67,108,864 ✓ (correct if meant as kB)

// defaultMaxMem in bytes:
// 64 GB = 64 * 1024 * 1024 * 1024 bytes = 68,719,476,736 bytes
// 64 << 30 = 68,719,476,736 ← current value (wrong if API returns kB)
```

## Suggested Fix

```go
const defaultMaxMem = 64 << 20 // 67108864 kB = 64 GiB (was 64 << 30)
```
