package main

import (
	"fmt"
	"time"

	"golang.org/x/exp/rand"
)

// This gives  delay -- shows how long it takes to run by sleeping for a random number between 0 and 5 seconds in in the sleepMe. This should take a while to run.
// To keep the code simple use the 'time' command to run it:
// time go run x-concurrent.go
// Using go routines this version is done in just over 4 seconds -- with a max sleep time of 3 seconds. 1 for compile, 3 seconds to run, not bad.

func main() {
	// Think of go routines as balls, you toss 10 of them in the air, well you have to catch 10 balls in return. Setup a channel that can receive the results from the go routines.
	// Make a channel for a string.
	sleepChan := make(chan string)
	for i := 1; i <= 10; i++ {
		// Toss ball in the air.
		go sleepMe(i, sleepChan)
	}
	fmt.Println("Done Spinnup of Go Routines.")
	// catch the balls here, the first loop tossed 10 balls up, need to loop 10 times to catch them.
	for i := 1; i <= 10; i++ {
		// Just wait here to catch the ball, wait for ANY of the go routines to send you it's result.
		out := <-sleepChan
		// print what was caught off the sleepChan
		fmt.Print(out)
	}
}

// Add the sleepChan to the variables passed, remove the return as it will send the results back on the channel instead of the return
func sleepMe(number int, sleepChan chan string) {
	min := 1
	max := 5
	// Think of sleeptime as the how high we're going to throw the ball. The higher it goes the longer it takes to come down.
	sleeptime := rand.Intn(max - min)
	timeDir := time.Duration(sleeptime)
	time.Sleep(timeDir * time.Second)
	sleepChan <- fmt.Sprintf("%d is Done, Slept for %d second(s)\n", number, sleeptime)
}

/* Notice: it doesn't return in order. This shows concurrency, the first routine launched had a sleep time of 3 seconds,
and ws the last to finish. 10 which was launched last but had a sleep time of 1 finished much sooner.
$ time go run 2-concurrent.go
  Done Spinnup of Go Routines.
  2 is Done, Slept for 0 second(s)
  7 is Done, Slept for 0 second(s)
  6 is Done, Slept for 1 second(s)
  10 is Done, Slept for 1 second(s)
  5 is Done, Slept for 1 second(s)
  9 is Done, Slept for 2 second(s)
  4 is Done, Slept for 2 second(s)
  3 is Done, Slept for 3 second(s)
  8 is Done, Slept for 3 second(s)
  1 is Done, Slept for 3 second(s)

  real	0m4.165s
  user	0m0.196s
  sys	0m0.273s
*/
