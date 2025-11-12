package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Simple test app started")

	counter := 0
	for {
		counter++
		processLoop(counter)
		time.Sleep(2 * time.Second)
	}
}

func processLoop(iteration int) {
	message := fmt.Sprintf("Iteration %d", iteration)
	result := calculate(iteration)
	fmt.Printf("%s: result = %d\n", message, result)
}

func calculate(n int) int {
	return n * n
}
