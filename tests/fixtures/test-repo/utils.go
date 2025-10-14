package main

import (
	"errors"
	"math"
)

// StringHelper provides string utility functions
type StringHelper interface {
	Reverse(s string) string
	Length(s string) int
}

// MathUtils provides mathematical utility functions
type MathUtils struct{}

// Square returns the square of a number
func (m *MathUtils) Square(x float64) float64 {
	return math.Pow(x, 2)
}

// Sqrt returns the square root of a number
func (m *MathUtils) Sqrt(x float64) (float64, error) {
	if x < 0 {
		return 0, errors.New("cannot calculate square root of negative number")
	}
	return math.Sqrt(x), nil
}

// Constants for configuration
const (
	MaxRetries = 3
	Timeout    = 30
)
