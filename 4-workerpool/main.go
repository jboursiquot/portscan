package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var host string
var ports string
var numWorkers int

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan.")
	flag.StringVar(&ports, "ports", "80", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of workers. Defaults to 10.")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("Failed to parse ports to scan: %s", err)
		os.Exit(1)
	}

	portsChan := make(chan int, numWorkers)
	resultsChan := make(chan int)

	for i := 0; i < cap(portsChan); i++ { // numWorkers also acceptable here
		go worker(host, portsChan, resultsChan)
	}

	go func() {
		for _, p := range portsToScan {
			portsChan <- p
		}
	}()

	var openPorts []int
	for i := 0; i < len(portsToScan); i++ {
		if p := <-resultsChan; p != 0 { // non-zero port means it's open
			openPorts = append(openPorts, p)
		}
	}

	close(portsChan)
	close(resultsChan)

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

func worker(host string, portsChan <-chan int, resultsChan chan<- int) {
	for p := range portsChan {
		address := fmt.Sprintf("%s:%d", host, p)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			fmt.Printf("%d CLOSED (%s)\n", p, err)
			resultsChan <- 0
			continue
		}
		conn.Close()
		resultsChan <- p
	}
}
