package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var host string
var ports string
var batchSize int

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan.")
	flag.StringVar(&ports, "ports", "80", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&batchSize, "batchSize", 15, "Batch size of pipeline (default is 10).")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("Failed to parse ports to scan: %s\n", err)
		os.Exit(1)
	}

	dest, err := os.Create("scans-fan-out-fan-in.csv")
	if err != nil {
		fmt.Printf("Failed to create scan results destination: %s\n", err)
		os.Exit(2)
	}

	// fan out
	batches := batch(portsToScan, batchSize)
	var scanChans []<-chan scanOp
	for _, b := range batches {
		scanChans = append(scanChans, doScan(genScanChan(host, b...)))
	}

	scanChan := storeScan(
		dest,
		openPortsOnly(
			fanIn(scanChans...),
		),
	)
	for s := range scanChan {
		if !s.open && s.scanErr != fmt.Sprintf("dial tcp 127.0.0.1:%d: connect: connection refused", s.port) {
			fmt.Println(s.scanErr)
		}
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

type scanOp struct {
	host         string
	port         int
	open         bool
	scanErr      string
	scanDuration time.Duration
}

func (so scanOp) csvHeaders() []string {
	return []string{"host", "port", "open", "scanError", "scanDuration"}
}

func (so scanOp) asSlice() []string {
	return []string{
		so.host,
		strconv.FormatInt(int64(so.port), 10),
		strconv.FormatBool(so.open),
		so.scanErr,
		so.scanDuration.String(),
	}
}

func genScanChan(host string, ports ...int) <-chan scanOp {
	c := make(chan scanOp, len(ports))
	for _, p := range ports {
		c <- scanOp{host: host, port: p}
	}
	close(c)
	return c
}

func doScan(scans <-chan scanOp) <-chan scanOp {
	c := make(chan scanOp)
	go func() {
		defer close(c)
		for scan := range scans {
			address := fmt.Sprintf("%s:%d", scan.host, scan.port)
			start := time.Now()
			conn, err := net.Dial("tcp", address)
			scan.scanDuration = time.Since(start)
			if err != nil {
				scan.scanErr = err.Error()
			} else {
				conn.Close()
				scan.open = true
			}
			c <- scan
		}
	}()
	return c
}

func openPortsOnly(scans <-chan scanOp) <-chan scanOp {
	c := make(chan scanOp)
	go func() {
		defer close(c)
		for scan := range scans {
			if scan.open {
				c <- scan
			}
		}
	}()
	return c
}

func storeScan(file io.Writer, scans <-chan scanOp) <-chan scanOp {
	w := csv.NewWriter(file)
	c := make(chan scanOp)
	go func() {
		defer w.Flush()
		defer close(c)
		var headerWritten bool
		for scan := range scans {
			if !headerWritten {
				headers := scan.csvHeaders()
				if err := w.Write(headers); err != nil {
					fmt.Println(err)
					break
				}
				headerWritten = true
			}
			values := scan.asSlice()
			if err := w.Write(values); err != nil {
				fmt.Println(err)
				break
			}
			c <- scan
		}
	}()
	return c
}

func fanIn(scanChans ...<-chan scanOp) <-chan scanOp {
	c := make(chan scanOp)
	wg := sync.WaitGroup{}
	wg.Add(len(scanChans))

	for _, scanChan := range scanChans {
		go func(scanChan <-chan scanOp) {
			for scan := range scanChan {
				c <- scan
			}
			wg.Done()
		}(scanChan)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	return c
}

func batch(ports []int, size int) [][]int {
	var batches [][]int
	for i := 0; i < len(ports); i += size {
		end := i + size
		if end > len(ports) {
			end = len(ports)
		}
		batches = append(batches, ports[i:end])
	}
	return batches
}
