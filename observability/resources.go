package observability

import "runtime"

type ResourceSnapshot struct {
	HeapAllocBytes  uint64
	HeapSysBytes    uint64
	TotalAllocBytes uint64
	NumGC           uint32
	Goroutines      int
}

type ResourceUsage struct {
	HeapAllocBytes       uint64
	HeapAllocDeltaBytes  int64
	HeapSysBytes         uint64
	TotalAllocDeltaBytes uint64
	GCDelta              uint32
	Goroutines           int
	GoroutinesDelta      int
}

func CaptureResources() ResourceSnapshot {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return ResourceSnapshot{
		HeapAllocBytes:  stats.HeapAlloc,
		HeapSysBytes:    stats.HeapSys,
		TotalAllocBytes: stats.TotalAlloc,
		NumGC:           stats.NumGC,
		Goroutines:      runtime.NumGoroutine(),
	}
}

func (before ResourceSnapshot) Diff(after ResourceSnapshot) ResourceUsage {
	return ResourceUsage{
		HeapAllocBytes:       after.HeapAllocBytes,
		HeapAllocDeltaBytes:  int64(after.HeapAllocBytes) - int64(before.HeapAllocBytes),
		HeapSysBytes:         after.HeapSysBytes,
		TotalAllocDeltaBytes: uint64Delta(before.TotalAllocBytes, after.TotalAllocBytes),
		GCDelta:              uint32Delta(before.NumGC, after.NumGC),
		Goroutines:           after.Goroutines,
		GoroutinesDelta:      after.Goroutines - before.Goroutines,
	}
}

func uint64Delta(before, after uint64) uint64 {
	if after < before {
		return 0
	}

	return after - before
}

func uint32Delta(before, after uint32) uint32 {
	if after < before {
		return 0
	}

	return after - before
}
