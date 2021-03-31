package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	for i := 5200; i <= 5500; i++ {
		go func(p int) {
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", p))
			if err != nil {
				log.Printf("%d CLOSED (%s)\n", p, err)
				return
			}
			conn.Close()
			log.Printf("%d OPEN\n", p)
		}(i)
	}
	log.Println("DONE")
}
