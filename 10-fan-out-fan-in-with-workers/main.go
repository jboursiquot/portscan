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
	flag.StringVar(&ports, "ports", "5400-5500", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "Number of workers (defaults to # of logical CPUs).")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("Failed to parse ports to scan: %s\n", err)
		os.Exit(1)
	}

	// The done channel will be shared by the entire pipeline
	// so that when it's closed it serves as a signal
	// for all the goroutines we started to exit.
	done := make(chan struct{})
	defer close(done)

	in := gen(done, portsToScan...)

	// fan-out
	var chans []<-chan scanOp
	for i := 0; i < workers; i++ {
		chans = append(chans, scan(done, in))
	}

	// for s := range filterOpen(done, merge(done, chans...)) {
	// 	fmt.Printf("%#v\n", s)
	// }

	for s := range filterErr(done, merge(done, chans...)) {
		fmt.Printf("%#v\n", s)
		done <- struct{}{}
		return
	}

	// done chan is closed by the deferred call here
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

func gen(done <-chan struct{}, ports ...int) <-chan scanOp {
	out := make(chan scanOp, len(ports))
	go func() {
		defer close(out)
		for _, p := range ports {
			select {
			case out <- scanOp{port: p}:
			case <-done:
				return
			}
		}
	}()
	return out
}

func scan(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
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
			case <-done:
				return
			}
		}
	}()
	return out
}

func filterOpen(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
				if scan.open {
					out <- scan
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

func filterErr(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
				if !scan.open && strings.Contains(scan.scanErr, "too many open files") {
					out <- scan
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

func merge(done <-chan struct{}, chans ...<-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	wg := sync.WaitGroup{}
	wg.Add(len(chans))

	for _, sc := range chans {
		go func(sc <-chan scanOp) {
			defer wg.Done()
			for scan := range sc {
				select {
				case out <- scan:
				case <-done:
					return
				}
			}
		}(sc)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
