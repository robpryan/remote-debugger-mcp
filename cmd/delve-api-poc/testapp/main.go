package main

import (
	"fmt"
	"time"
)

// Simple test application for debugging
func main() {
	fmt.Println("Test application starting...")

	counter := 0
	message := "Hello from debugger test!"

	for i := 0; i < 10; i++ {
		counter++
		processIteration(i, message)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("Test application finished!")
}

func processIteration(iteration int, msg string) {
	result := calculate(iteration)
	fmt.Printf("Iteration %d: %s (result: %d)\n", iteration, msg, result)
}

func calculate(n int) int {
	if n == 0 {
		return 1
	}
	return n * calculate(n-1)
}
