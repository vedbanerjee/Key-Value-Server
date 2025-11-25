package main

import (
    "context"
    "fmt"
    "io"
    "net"
    "net/http"
    "sync"
    "sync/atomic"
    "time"
    "runtime"
    "flag"
)

func main() {
    var (
        urlFlag        = flag.String("url", "http://localhost:8080/get", "request BASE URL (key will be added as ?key=X)")
        threadsFlag    = flag.Int("threads", 10, "number of concurrent client workers (closed-loop users)")
        durationFlag   = flag.Duration("duration", 300*time.Second, "test duration (e.g. 300s)")
        reqTimeoutFlag = flag.Duration("reqtimeout", 5*time.Second, "per-request timeout")
        keyFlag        = flag.Int("key", 5, "fixed key value to use in the URL query parameter (?key=X)")
    )
    flag.Parse()

    fmt.Printf("Load generator\n  URL=%s?key=%d\n  threads=%d\n  duration=%s\n  Timeout=%s\n\n",
        *urlFlag, *keyFlag, *threadsFlag, durationFlag.String(), reqTimeoutFlag.String())

    runtime.GOMAXPROCS(runtime.NumCPU())

    transport := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   5 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          1000,
        MaxIdleConnsPerHost:   1000,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   5 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   *reqTimeoutFlag,
    }

    var totalRequests uint64
    var totalSuccess uint64
    var totalErrors uint64
    var totalLatencyNs uint64 

    requestURL := fmt.Sprintf("%s?key=%d", *urlFlag, *keyFlag)

    stop := make(chan struct{})

    var wg sync.WaitGroup
    wg.Add(*threadsFlag)

    startTime := time.Now()
    for i := 0; i < *threadsFlag; i++ {
        go func(workerID int) {
            defer wg.Done()
            for {
                select {
                case <-stop:
                    return
                default:
                    ctx, cancel := context.WithTimeout(context.Background(), *reqTimeoutFlag)

                    req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
                    if err != nil {
                        atomic.AddUint64(&totalErrors, 1)
                        atomic.AddUint64(&totalRequests, 1)
                        cancel()
                        time.Sleep(5 * time.Millisecond)
                        continue
                    }

                    t0 := time.Now()
                    resp, err := client.Do(req)
                    lat := time.Since(t0)
                    atomic.AddUint64(&totalRequests, 1)
                    atomic.AddUint64(&totalLatencyNs, uint64(lat.Nanoseconds()))

                    if err != nil {
                        atomic.AddUint64(&totalErrors, 1)
                        cancel()
                        time.Sleep(10 * time.Millisecond)
                        continue
                    }

                    _, _ = io.ReadAll(resp.Body)
                    resp.Body.Close()

                    if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
                        atomic.AddUint64(&totalSuccess, 1)
                    } else {
                        atomic.AddUint64(&totalErrors, 1)
                    }
                    cancel()
                }
            }
        }(i)
    }

    time.Sleep(*durationFlag)
    close(stop)
    wg.Wait()

    elapsed := time.Since(startTime)
    totalReq := atomic.LoadUint64(&totalRequests)
    successReq := atomic.LoadUint64(&totalSuccess)
    errReq := atomic.LoadUint64(&totalErrors)
    sumLatencyNs := atomic.LoadUint64(&totalLatencyNs)

    throughput := float64(successReq) / elapsed.Seconds()
    avgLatencyMs := 0.0
    if totalReq > 0 {
        avgLatencyMs = float64(sumLatencyNs) / float64(totalReq) / 1e6
    }

    fmt.Println("Load test summary")
    fmt.Printf("Duration: %s\n", elapsed)
    fmt.Printf("Threads: %d\n", *threadsFlag)
    fmt.Printf("Total requests: %d (success=%d, errors=%d)\n", totalReq, successReq, errReq)
    fmt.Printf("Throughput (successful req/s): %.2f\n", throughput)
    fmt.Printf("Average latency (ms, across all requests): %.3f\n", avgLatencyMs)
    fmt.Printf("Requests/sec (all): %.2f\n", float64(totalReq)/elapsed.Seconds())
    fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
}
