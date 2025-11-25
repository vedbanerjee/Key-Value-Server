package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type keyValue struct {
	Key   int    `json:"key"`
	Value string `json:"value"`
}

func main() {
	var (
		urlFlag      = flag.String("url", "http://localhost:8080/put", "PUT URL")
		threadsFlag  = flag.Int("threads", 10, "number of concurrent worker threads")
		durationFlag = flag.Duration("duration", 60*time.Second, "test duration")
		keyCountFlag = flag.Int("keycount", 100, "number of unique keys to randomly choose from")
		timeoutFlag  = flag.Duration("timeout", 5*time.Second, "per-request timeout")
	)
	flag.Parse()

	if *keyCountFlag <= 0 {
		fmt.Println("keycount must be > 0")
		os.Exit(1)
	}

	fmt.Printf("\nLoad Generator\n")
	fmt.Printf("URL: %s\nThreads: %d\nDuration: %s\nKeyCount: %d\nTimeout: %s\n\n",
		*urlFlag, *threadsFlag, durationFlag.String(), *keyCountFlag, timeoutFlag.String())

	runtime.GOMAXPROCS(runtime.NumCPU())

	jsonBodies := make([][]byte, *keyCountFlag)
	for i := 1; i <= *keyCountFlag; i++ {
		obj := keyValue{Key: i, Value: fmt.Sprintf("%d", i)}
		b, _ := json.Marshal(obj)
		jsonBodies[i-1] = b
	}

	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   false,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   *timeoutFlag,
	}

	var total uint64
	var success uint64
	var errors uint64
	var latencyNs uint64

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(*threadsFlag)

	for w := 0; w < *threadsFlag; w++ {
		go func(id int) {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
					keyIdx := time.Now().UnixNano() % int64(*keyCountFlag)
					body := jsonBodies[keyIdx]

					ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
					req, err := http.NewRequestWithContext(ctx, "PUT", *urlFlag, bytes.NewReader(body))
					if err != nil {
						atomic.AddUint64(&errors, 1)
						atomic.AddUint64(&total, 1)
						cancel()
						continue
					}
					req.Header.Set("Content-Type", "application/json")

					start := time.Now()
					resp, err := client.Do(req)
					elapsed := time.Since(start)

					atomic.AddUint64(&total, 1)
					atomic.AddUint64(&latencyNs, uint64(elapsed.Nanoseconds()))

					if err != nil {
						atomic.AddUint64(&errors, 1)
						cancel()
						continue
					}

					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						atomic.AddUint64(&success, 1)
					} else {
						atomic.AddUint64(&errors, 1)
					}

					cancel()
				}
			}
		}(w)
	}

	time.Sleep(*durationFlag)
	close(stop)
	wg.Wait()

	totalReq := atomic.LoadUint64(&total)
	successReq := atomic.LoadUint64(&success)
	errorReq := atomic.LoadUint64(&errors)
	avgLatency := float64(atomic.LoadUint64(&latencyNs)) / float64(totalReq) / 1e6
	throughput := float64(successReq) / durationFlag.Seconds()

	fmt.Println("\nLoad Test Summary")
	fmt.Printf("Duration: %s\n", durationFlag.String())
	fmt.Printf("Threads: %d\n", *threadsFlag)
	fmt.Printf("Total Requests: %d\n", totalReq)
	fmt.Printf("Success: %d\nErrors: %d\n", successReq, errorReq)
	fmt.Printf("Throughput (successful req/s): %.2f\n", throughput)
	fmt.Printf("Average Latency: %.3f ms\n", avgLatency)
	fmt.Printf("Requests/sec (all): %.2f\n", float64(totalReq)/durationFlag.Seconds())
}
