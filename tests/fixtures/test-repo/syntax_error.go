//go:build syntax_error_test

package main

// This file has intentional syntax errors for testing error handling

func BrokenFunction() {
	// Missing closing brace
	if true {
		x := 10
	// Missing closing brace for function
