package main

import (
	"fmt"
	"strings"
)

// Main entry point for the test application
func main() {
	result := ProcessData("hello world")
	fmt.Println(result)
}

// ProcessData processes input string and returns formatted output
func ProcessData(input string) string {
	return strings.ToUpper(input)
}

// Calculator provides basic arithmetic operations
type Calculator struct {
	value int
}

// NewCalculator creates a new Calculator instance
func NewCalculator(initial int) *Calculator {
	return &Calculator{value: initial}
}

// Add adds a number to the calculator value
func (c *Calculator) Add(n int) int {
	c.value += n
	return c.value
}

// Multiply multiplies the calculator value by n
func (c *Calculator) Multiply(n int) int {
	c.value *= n
	return c.value
}
