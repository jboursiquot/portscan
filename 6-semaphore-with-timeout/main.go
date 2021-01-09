package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

var host string
var ports string
var numWorkers int
var timeout int

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan.")
	flag.StringVar(&ports, "ports", "80", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of workers. Defaults to system's number of CPUs.")
	flag.IntVar(&timeout, "timeout", 5, "Timeout in seconds (default is 5).")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("Failed to parse ports to scan: %s", err)
		os.Exit(1)
	}

	sem := semaphore.NewWeighted(int64(numWorkers))
	openPorts := make([]int, 0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	for _, port := range portsToScan {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v", err)
			break
		}

		go func(port int) {
			defer sem.Release(1)
			sleepy(10)
			p := scan(host, port)
			if p != 0 {
				openPorts = append(openPorts, p)
			}
		}(port)
	}

	if err := sem.Acquire(ctx, int64(numWorkers)); err != nil {
		fmt.Printf("Failed to acquire semaphore: %v", err)
	}

	sort.Ints(openPorts)
	for _, p := range openPorts {
		fmt.Printf("%d - open\n", p)
	}
}

func parsePortsToScan(portsFlag string) ([]int, error) {
	p, err := strconv.Atoi(portsFlag)
	if err == nil {
		return []int{p}, nil
	}

	ports := strings.Split(portsFlag, "-")
	if len(ports) != 2 {
		return nil, errors.New("unable to determine port(s) to scan")
	}

	minPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[0])
	}

	maxPort, err := strconv.Atoi(ports[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[1])
	}

	if minPort <= 0 || maxPort <= 0 {
		return nil, fmt.Errorf("port numbers must be greater than 0")
	}

	var results []int
	for p := minPort; p <= maxPort; p++ {
		results = append(results, p)
	}
	return results, nil
}

func scan(host string, port int) int {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("%d CLOSED (%s)\n", port, err)
		return 0
	}
	conn.Close()
	return port
}

func sleepy(max int) {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(max)
	time.Sleep(time.Duration(n) * time.Second)
}
