package paymenthandler

import (
	"fmt"
	"os"
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
				Requests: []struct {
					TransactionID string
					PaymentID     string
					Type          string
				}{
					{
						TransactionID: fmt.Sprintf("%d", i),
						PaymentID:     fmt.Sprintf("%d", i),
						Type:          "resolve",
					},
					{
						TransactionID: fmt.Sprintf("%d", i+1),
						PaymentID:     fmt.Sprintf("%d", i+1),
						Type:          "resolve",
					},
				},
			},
		}
	}

	file, err := os.Create("resolver_benchmark_stats_parallel.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer file.Close()

	b.Run("Parallel", func(b *testing.B) {
		b.SetParallelism(10) // Set the level of parallelism
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for i, bm := range benchmarks {
					start := time.Now()
					_, err := resolveRequest("localhost:3000", bm.data)
					if err != nil {
						b.Fatal(err)
					}
					elapsed := time.Since(start).Microseconds()
					fmt.Fprintf(file, "Call %d: %d microseconds\n", i+1, elapsed)
				}
			}
		})
	})
}
