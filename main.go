package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL        = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval     = 5 * time.Second
	maxErrorCount    = 3
	loadAvgThreshold = 30.0
	memThreshold     = 0.80
	diskThreshold    = 0.90
	netThreshold     = 0.90
)

func main() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	errorCounter := 0

	for {
		err := checkServerStats(client)
		if err != nil {
			errorCounter++
			if errorCounter >= maxErrorCount {
				fmt.Println("Unable to fetch server statistic")
			}
		} else {
			errorCounter = 0
		}

		time.Sleep(pollInterval)
	}
}

func checkServerStats(client *http.Client) error {
	resp, err := client.Get(serverURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dataStr := strings.TrimSpace(string(body))
	parts := strings.Split(dataStr, ",")

	if len(parts) != 7 {
		return fmt.Errorf("invalid data format: expected 7 values, got %d", len(parts))
	}

	stats, err := parseStats(parts)
	if err != nil {
		return err
	}

	analyzeStats(stats)

	return nil
}

type ServerStats struct {
	LoadAvg   float64
	TotalMem  float64
	UsedMem   float64
	TotalDisk float64
	UsedDisk  float64
	TotalNet  float64
	UsedNet   float64
}

func parseStats(parts []string) (ServerStats, error) {
	var s ServerStats
	var err error

	parseFloat := func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}

	if s.LoadAvg, err = parseFloat(parts[0]); err != nil {
		return s, err
	}
	if s.TotalMem, err = parseFloat(parts[1]); err != nil {
		return s, err
	}
	if s.UsedMem, err = parseFloat(parts[2]); err != nil {
		return s, err
	}
	if s.TotalDisk, err = parseFloat(parts[3]); err != nil {
		return s, err
	}
	if s.UsedDisk, err = parseFloat(parts[4]); err != nil {
		return s, err
	}
	if s.TotalNet, err = parseFloat(parts[5]); err != nil {
		return s, err
	}
	if s.UsedNet, err = parseFloat(parts[6]); err != nil {
		return s, err
	}

	return s, nil
}

func analyzeStats(s ServerStats) {
	if s.LoadAvg > loadAvgThreshold {
		fmt.Printf("Load Average is too high: %.0f\n", s.LoadAvg)
	}

	if s.TotalMem > 0 {
		memUsagePercent := (s.UsedMem / s.TotalMem) * 100
		if (s.UsedMem / s.TotalMem) > memThreshold {
			fmt.Printf("Memory usage too high: %.0f%%\n", memUsagePercent)
		}
	}

	if s.TotalDisk > 0 {
		if (s.UsedDisk / s.TotalDisk) > diskThreshold {
			freeBytes := s.TotalDisk - s.UsedDisk
			freeMb := int64(freeBytes) / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeMb)
		}
	}

	if s.TotalNet > 0 {
		if (s.UsedNet / s.TotalNet) > netThreshold {
			freeBps := s.TotalNet - s.UsedNet
			freeMbit := int64(freeBps) / 1000000
			fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", freeMbit)
		}
	}
}
