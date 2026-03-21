package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	host       = "http://127.0.0.1:1133"
	token      = "abc"
	concurrent = 20  // concurrent captures
	total      = 100 // total captures to send
)

type CaptureRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type CaptureResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	NotePath string `json:"note_path"`
	Error    string `json:"error"`
	Code     string `json:"code"`
}

type Result struct {
	id       int
	success  bool
	notePath string
	err      string
	duration time.Duration
}

func capture(id int, content string) Result {
	start := time.Now()

	body, _ := json.Marshal(CaptureRequest{
		Type:    "text",
		Content: content,
	})

	req, _ := http.NewRequest("POST", host+"/v1/capture", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Khayal-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{id: id, success: false, err: err.Error(), duration: time.Since(start)}
	}
	defer resp.Body.Close()

	var r CaptureResponse
	json.NewDecoder(resp.Body).Decode(&r)

	if (resp.StatusCode != 200 && resp.StatusCode != 201) || r.Error != "" {
		return Result{id: id, success: false, err: fmt.Sprintf("%d %s %s", resp.StatusCode, r.Code, r.Error), duration: time.Since(start)}
	}

	return Result{id: id, success: true, notePath: r.NotePath, duration: time.Since(start)}
}

func main() {
	fmt.Printf("stress test · %d concurrent · %d total\n\n", concurrent, total)

	sem := make(chan struct{}, concurrent)
	results := make([]Result, total)
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(id int) {
			defer wg.Done()
			defer func() { <-sem }()
			content := fmt.Sprintf("stress test capture %d at %d", id, time.Now().UnixNano())
			results[id] = capture(id, content)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// ── Results ───────────────────────────────────────────────────
	succeeded := 0
	failed := 0
	notePaths := make(map[string][]int)
	var durations []time.Duration

	for _, r := range results {
		if r.success {
			succeeded++
			notePaths[r.notePath] = append(notePaths[r.notePath], r.id)
		} else {
			failed++
			fmt.Printf("  ✗ capture %d failed: %s\n", r.id, r.err)
		}
		durations = append(durations, r.duration)
	}

	// check for duplicate note paths
	duplicates := 0
	for path, ids := range notePaths {
		if len(ids) > 1 {
			duplicates++
			fmt.Printf("  ✗ duplicate note path: %s → captures %v\n", path, ids)
		}
	}

	// latency stats
	var totalMs int64
	var maxMs int64
	for _, d := range durations {
		ms := d.Milliseconds()
		totalMs += ms
		if ms > maxMs {
			maxMs = ms
		}
	}
	avgMs := totalMs / int64(len(durations))

	fmt.Printf("results\n")
	fmt.Printf("  total      %d\n", total)
	fmt.Printf("  succeeded  %d\n", succeeded)
	fmt.Printf("  failed     %d\n", failed)
	fmt.Printf("  duplicates %d\n", duplicates)
	fmt.Printf("  elapsed    %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  avg        %dms\n", avgMs)
	fmt.Printf("  max        %dms\n", maxMs)
	fmt.Printf("  throughput %.1f req/s\n", float64(total)/elapsed.Seconds())
	fmt.Println()

	// ── Wait for worker to process all jobs ───────────────────────
	fmt.Printf("waiting for worker to process %d jobs...\n", succeeded)
	waitForQueue()

	// ── Verify vault ──────────────────────────────────────────────
	verifyVault(notePaths)

	if failed > 0 || duplicates > 0 {
		fmt.Println("\n✗ stress test FAILED")
		os.Exit(1)
	}
	fmt.Println("\n✓ stress test PASSED")
}

func waitForQueue() {
	client := &http.Client{}
	deadline := time.Now().Add(5 * time.Minute)

	for time.Now().Before(deadline) {
		// Check pending
		req, _ := http.NewRequest("GET", host+"/v1/queue?status=pending&limit=1", nil)
		req.Header.Set("X-Khayal-Token", token)
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		var result struct {
			Total int `json:"total"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		// Check queued
		req2, _ := http.NewRequest("GET", host+"/v1/queue?status=queued&limit=1", nil)
		req2.Header.Set("X-Khayal-Token", token)
		resp2, _ := client.Do(req2)
		var result2 struct {
			Total int `json:"total"`
		}
		json.NewDecoder(resp2.Body).Decode(&result2)
		resp2.Body.Close()

		// Check processing
		req3, _ := http.NewRequest("GET", host+"/v1/queue?status=processing&limit=1", nil)
		req3.Header.Set("X-Khayal-Token", token)
		resp3, _ := client.Do(req3)
		var result3 struct {
			Total int `json:"total"`
		}
		json.NewDecoder(resp3.Body).Decode(&result3)
		resp3.Body.Close()

		if result.Total == 0 && result2.Total == 0 && result3.Total == 0 {
			fmt.Printf("  ✓ queue empty\n")
			return
		}

		fmt.Printf("  pending: %d  queued: %d  processing: %d\r", result.Total, result2.Total, result3.Total)
		time.Sleep(2 * time.Second)
	}
	fmt.Println("  ✗ timed out waiting for queue")
}

func verifyVault(notePaths map[string][]int) {
	fmt.Println("verifying vault...")
	missing := 0
	for path := range notePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  ✗ missing note: %s\n", path)
			missing++
		}
	}
	if missing == 0 {
		fmt.Printf("  ✓ all %d notes present in vault\n", len(notePaths))
	}
}
