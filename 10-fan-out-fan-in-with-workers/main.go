package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ports string
var workers int

func init() {
	flag.StringVar(&ports, "ports", "80", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "Number of workers (defaults to # of logical CPUs).")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("Failed to parse ports to scan: %s\n", err)
		os.Exit(1)
	}

	in := gen(portsToScan...)

	// fan-out
	var chans []<-chan scanOp
	for i := 0; i < workers; i++ {
		chans = append(chans, scan(in))
	}

	for s := range filterOpen(merge(chans...)) {
		fmt.Printf("%#v\n", s)
	}

	// for s := range filterErr(merge(chans...)) {
	// 	fmt.Printf("%#v\n", s)
	// }
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

type scanOp struct {
	port         int
	open         bool
	scanErr      string
	scanDuration time.Duration
}

func gen(ports ...int) <-chan scanOp {
	out := make(chan scanOp, len(ports))
	for _, p := range ports {
		out <- scanOp{port: p}
	}
	close(out)
	return out
}

func scan(in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			address := fmt.Sprintf("127.0.0.1:%d", scan.port)
			start := time.Now()
			conn, err := net.Dial("tcp", address)
			scan.scanDuration = time.Since(start)
			if err != nil {
				scan.scanErr = err.Error()
			} else {
				conn.Close()
				scan.open = true
			}
			out <- scan
		}
	}()
	return out
}

func filterOpen(in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			if scan.open {
				out <- scan
			}
		}
	}()
	return out
}

func filterErr(in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			if !scan.open && strings.Contains(scan.scanErr, "too many open files") {
				out <- scan
			}
		}
	}()
	return out
}

func merge(chans ...<-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	wg := sync.WaitGroup{}
	wg.Add(len(chans))

	for _, sc := range chans {
		go func(sc <-chan scanOp) {
			for scan := range sc {
				out <- scan
			}
			wg.Done()
		}(sc)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
