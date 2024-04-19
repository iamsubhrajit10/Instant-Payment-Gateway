package paymenthandler

import (
	"fmt"
	"os"
	"sort"
	"testing"
	"time"
)

func BenchmarkResolveRequest(b *testing.B) {
	benchmarks := make([]struct {
		name string
		data RequestDataResolver
	}, 100000)

	for i := 1; i <= 100000; i++ {
		benchmarks[i-1] = struct {
			name string
			data RequestDataResolver
		}{
			fmt.Sprintf("Benchmark%d", i),
			RequestDataResolver{
				PaymentID:     fmt.Sprintf("%d", i),
				TransactionID: fmt.Sprintf("%d", i),
				Type:          "resolve",
			},
		}
	}

	times := make([]int64, len(benchmarks))

	b.Run("Parallel", func(b *testing.B) {
		b.SetParallelism(4) // Set the level of parallelism
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for i, bm := range benchmarks {
					start := time.Now()
					_, err := resolveRequest("localhost:3000", bm.data)
					if err != nil {
						b.Fatal(err)
					}
					times[i] = time.Since(start).Microseconds()
				}
			}
		})
	})

	// Calculate statistics
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	total := int64(0)
	for _, t := range times {
		total += t
	}
	avg := total / int64(len(times))
	min := times[0]
	max := times[len(times)-1]
	median := times[len(times)/2]

	// Prepare a file to write the statistics
	file, err := os.Create("benchmark_stats_parallel.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()

	// Write statistics to the file
	_, err = fmt.Fprintf(file, "Min: %d, Max: %d, Avg: %d, Median: %d\n", min, max, avg, median)
	if err != nil {
		b.Fatal(err)
	}
}
