package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Person represents a simple struct for testing
type Person struct {
	Name string
	Age  int
	City string
}

// Counter tracks values in goroutines
type Counter struct {
	mu    sync.Mutex
	value int
}

func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func main() {
	fmt.Println("=== Delve MCP Test Application ===")
	fmt.Println("This app runs indefinitely for debugging practice")
	fmt.Println()

	// Initialize some variables
	message := "Hello from test app!"
	numbers := []int{1, 2, 3, 4, 5}
	counter := &Counter{}

	// Create a person
	person := Person{
		Name: "Alice",
		Age:  30,
		City: "San Francisco",
	}

	fmt.Printf("Starting with: %s\n", message)
	fmt.Printf("Person: %+v\n", person)
	fmt.Printf("Numbers: %v\n", numbers)
	fmt.Println()

	// Start some goroutines (all run forever)

	// Goroutine 1: Counter incrementer
	go func() {
		for {
			counter.Increment()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Goroutine 2: Number processor (runs forever)
	go func() {
		for {
			processNumbers(numbers)
			time.Sleep(3 * time.Second)
		}
	}()

	// Goroutine 3: Person updater (runs forever)
	go func() {
		for {
			updatePerson(&person)
			time.Sleep(2 * time.Second)
		}
	}()

	// Main loop
	iteration := 0
	for {
		iteration++

		// Call various functions to test breakpoints
		result := calculateSum(numbers)
		factorial := calculateFactorial(5)
		isPrime := checkPrime(17)

		fmt.Printf("\n[Iteration %d]\n", iteration)
		fmt.Printf("  Sum of numbers: %d\n", result)
		fmt.Printf("  Factorial(5): %d\n", factorial)
		fmt.Printf("  Is 17 prime? %v\n", isPrime)
		fmt.Printf("  Counter value: %d\n", counter.Value())
		fmt.Printf("  Person: %s, Age: %d\n", person.Name, person.Age)

		// Demonstrate local variables
		demonstrateLocalVars()

		time.Sleep(2 * time.Second)
	}

	// This will never be reached - program runs until killed
}

// calculateSum adds all numbers in a slice
func calculateSum(numbers []int) int {
	sum := 0
	for _, num := range numbers {
		sum += num
	}
	return sum
}

// calculateFactorial computes factorial recursively
func calculateFactorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * calculateFactorial(n-1)
}

// checkPrime checks if a number is prime
func checkPrime(n int) bool {
	if n <= 1 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := 3; i*i <= n; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// processNumbers simulates number processing
func processNumbers(numbers []int) {
	for i := 0; i < 20; i++ {
		total := 0
		for _, num := range numbers {
			total += num * 2
		}
		fmt.Printf("  [Goroutine] Processed numbers, total: %d\n", total)
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	}
}

// updatePerson simulates updating person data
func updatePerson(p *Person) {
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}
	for i := 0; i < 20; i++ {
		p.Age++
		p.City = cities[rand.Intn(len(cities))]
		fmt.Printf("  [Goroutine] Updated person: %s is now %d in %s\n", p.Name, p.Age, p.City)
		time.Sleep(time.Duration(rand.Intn(1500)) * time.Millisecond)
	}
}

// demonstrateLocalVars shows various local variable types
func demonstrateLocalVars() {
	// Different variable types for inspection
	intVar := 42
	stringVar := "test string"
	boolVar := true
	floatVar := 3.14159
	sliceVar := []string{"apple", "banana", "cherry"}
	mapVar := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	// Nested struct
	type Config struct {
		Timeout  int
		MaxRetry int
		Debug    bool
	}

	config := Config{
		Timeout:  30,
		MaxRetry: 3,
		Debug:    true,
	}

	// Use variables to prevent compiler optimization
	_ = intVar
	_ = stringVar
	_ = boolVar
	_ = floatVar
	_ = sliceVar
	_ = mapVar
	_ = config
}
