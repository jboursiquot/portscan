package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	p := 5000
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
	if err != nil {
		log.Fatalf("%d CLOSED (%s)\n", p, err)
	}
	conn.Close()
	log.Printf("%d OPEN\n", p)
}
