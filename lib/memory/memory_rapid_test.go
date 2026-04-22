package memory_test

import (
	"testing"

	"github.com/openGemini/openGemini/lib/memory"
	"pgregory.net/rapid"
)

func TestSysMem_NonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total, available := memory.GetMemMonitor().SysMem()
		if total < 0 {
			t.Fatalf("SysMem total = %d; want >= 0", total)
		}
		if available < 0 {
			t.Fatalf("SysMem available = %d; want >= 0", available)
		}
	})
}

func TestSysMem_TotalGeAvailable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total, available := memory.GetMemMonitor().SysMem()
		if total < available {
			t.Fatalf("SysMem total(%d) < available(%d)", total, available)
		}
	})
}

func TestMemUsedPct_Range(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pct := memory.GetMemMonitor().MemUsedPct()
		if pct < 0 || pct > 100 {
			t.Fatalf("MemUsedPct = %v; want [0, 100]", pct)
		}
	})
}

func TestSysMem_NonZero(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total, _ := memory.GetMemMonitor().SysMem()
		if total == 0 {
			t.Fatalf("SysMem total = 0")
		}
	})
}

func TestSysMem_ReasonableRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total, _ := memory.GetMemMonitor().SysMem()
		// On a real system, total should be in reasonable kB range
		// 1 MB = 1024 kB, 1 TB = 1024*1024*1024 kB
		if total > 1024*1024*1024 {
			t.Fatalf("SysMem total = %d kB = %.2f TB; seems too large (defaultMaxMem unit mismatch?)", total, float64(total)/1024/1024/1024)
		}
	})
}
