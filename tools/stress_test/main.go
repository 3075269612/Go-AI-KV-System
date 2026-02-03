package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	URL           = "http://localhost:8080/health"
	TotalRequests = 5000
	Concurrency   = 100 // Number of concurrent workers
)

func main() {
	var successCount int64
	var limitedCount int64
	var errorCount int64

	fmt.Printf("Starting stress test against %s\n", URL)
	fmt.Printf("Total Requests: %d, Concurrency: %d\n", TotalRequests, Concurrency)

	start := time.Now()
	var wg sync.WaitGroup
	requestsChan := make(chan struct{}, TotalRequests)

	// Fill the channel
	for i := 0; i < TotalRequests; i++ {
		requestsChan <- struct{}{}
	}
	close(requestsChan)

	// Start workers
	for i := 0; i < Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Timeout: 2 * time.Second,
			}
			for range requestsChan {
				resp, err := client.Get(URL)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else if resp.StatusCode == http.StatusTooManyRequests {
					atomic.AddInt64(&limitedCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("------------------------------------------------")
	fmt.Printf("Test Finished in %v\n", duration)
	fmt.Printf("Total Requests: %d\n", TotalRequests)
	fmt.Printf("Success (200): %d\n", successCount)
	fmt.Printf("Limited (429): %d\n", limitedCount)
	fmt.Printf("Errors:        %d\n", errorCount)
	fmt.Printf("Actual QPS:    %.2f\n", float64(TotalRequests)/duration.Seconds())
	fmt.Println("------------------------------------------------")
}
