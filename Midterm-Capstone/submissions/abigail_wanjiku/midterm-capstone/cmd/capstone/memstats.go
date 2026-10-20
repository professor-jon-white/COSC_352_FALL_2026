package main

import (
	"fmt"
	"io"
	"runtime"
	"runtime/metrics"
	"time"
)

// heapSampleEvery is how often the sampler reads the live heap size. The Go
// runtime does not record a peak, so sampling is the only way to observe it.
const heapSampleEvery = 5 * time.Millisecond

// memSampler records the largest heap seen while it runs.
type memSampler struct {
	stop chan struct{}
	done chan uint64
}

func startMemSampler() *memSampler {
	m := &memSampler{stop: make(chan struct{}), done: make(chan uint64, 1)}
	go func() {
		sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
		var peak uint64
		tick := time.NewTicker(heapSampleEvery)
		defer tick.Stop()
		for {
			metrics.Read(sample)
			peak = max(peak, sample[0].Value.Uint64())
			select {
			case <-m.stop:
				m.done <- peak
				return
			case <-tick.C:
			}
		}
	}()
	return m
}

// Stop ends sampling and returns the peak heap in bytes.
func (m *memSampler) Stop() uint64 {
	close(m.stop)
	return <-m.done
}

// printMemStats writes peak memory and GC activity for the whole process.
func printMemStats(w io.Writer, mode string, peakHeap uint64) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Fprintf(w, "\nMemory (%s mode)\n", mode)
	fmt.Fprintf(w, "  peak heap in use     %8s   (sampled every %v)\n", mib(peakHeap), heapSampleEvery)
	if rss, ok := peakRSS(); ok {
		fmt.Fprintf(w, "  peak process memory  %8s   (OS peak working set / max RSS)\n", mib(rss))
	}
	fmt.Fprintf(w, "  total allocated      %8s\n", mib(ms.TotalAlloc))
	fmt.Fprintf(w, "  heap objects alloc'd %8.1fM\n", float64(ms.Mallocs)/1e6)
	fmt.Fprintf(w, "  GC cycles            %8d\n", ms.NumGC)
	fmt.Fprintf(w, "  GC pause total       %8v\n", time.Duration(ms.PauseTotalNs).Round(10*time.Microsecond))
	fmt.Fprintf(w, "  GC CPU fraction      %7.1f%%\n", ms.GCCPUFraction*100)
}

func mib(b uint64) string { return fmt.Sprintf("%.0f MiB", float64(b)/(1<<20)) }
