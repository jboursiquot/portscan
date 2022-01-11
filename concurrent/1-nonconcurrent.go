package main

import (
	"fmt"
	"time"

	"golang.org/x/exp/rand"
)

// This gives  delay -- shows how long it takes to run by sleeping for a random number between 0 and 5 seconds in in the sleepMe. This should take a while to run.
// To keep the code simple use the 'time' command to run it:
// time go run 1-nonconcurrent.go
// real	0m23.431s <-- took 23 seconds to run on my system.

func main() {
	for i := 1; i <= 10; i++ {
		out := sleepMe(i)
		fmt.Print(out)
	}
}

func sleepMe(number int) string {
	min := 0
	max := 5
	sleeptime := rand.Intn(max - min)
	timeDir := time.Duration(sleeptime)
	time.Sleep(timeDir * time.Second)
	return fmt.Sprintf("%d is Done, Slept for %d second(s)\n", number, sleeptime)
}
