package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// CallAnalysisMetrics tracks precision and recall metrics for call analysis
type CallAnalysisMetrics struct {
	TruePositives  int // Correctly identified calls
	FalsePositives int // Incorrectly identified calls
	FalseNegatives int // Missed calls
	TrueNegatives  int // Correctly identified non-calls
}

// Precision calculates precision = TP / (TP + FP)
func (m *CallAnalysisMetrics) Precision() float64 {
	if m.TruePositives+m.FalsePositives == 0 {
		return 0
	}
	return float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
}

// Recall calculates recall = TP / (TP + FN)
func (m *CallAnalysisMetrics) Recall() float64 {
	if m.TruePositives+m.FalseNegatives == 0 {
		return 0
	}
	return float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)
}

// F1Score calculates F1 score = 2 * (Precision * Recall) / (Precision + Recall)
func (m *CallAnalysisMetrics) F1Score() float64 {
	p := m.Precision()
	r := m.Recall()
	if p+r == 0 {
		return 0
	}
	return 2 * (p * r) / (p + r)
}

// ExpectedCall represents an expected call relationship
type ExpectedCall struct {
	Source string
	Target string
}

// TestCallAnalysisMetrics_Go tests Go call analysis with precision/recall metrics
func TestCallAnalysisMetrics_Go(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "metrics.go")
	content := `package main

import "fmt"

func function1() {
	function2()
	function3()
	fmt.Println("test")
}

func function2() {
	function3()
	function4()
}

func function3() {
	function4()
}

func function4() {
	// leaf function
}

func isolatedFunction() {
	// does not call anything
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)

	// Define expected calls (ground truth)
	expectedCalls := []ExpectedCall{
		{"function1", "function2"},
		{"function1", "function3"},
		{"function1", "Println"},
		{"function2", "function3"},
		{"function2", "function4"},
		{"function3", "function4"},
	}

	// Extract actual calls
	actualCalls := make(map[string]bool)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			key := fmt.Sprintf("%s->%s", dep.Source, dep.Target)
			actualCalls[key] = true
		}
	}

	// Calculate metrics
	metrics := CallAnalysisMetrics{}

	for _, expected := range expectedCalls {
		key := fmt.Sprintf("%s->%s", expected.Source, expected.Target)
		if actualCalls[key] {
			metrics.TruePositives++
			delete(actualCalls, key) // Remove to track false positives
		} else {
			metrics.FalseNegatives++
			t.Logf("False Negative: %s -> %s", expected.Source, expected.Target)
		}
	}

	// Remaining actual calls are false positives
	metrics.FalsePositives = len(actualCalls)
	for key := range actualCalls {
		t.Logf("False Positive: %s", key)
	}

	// Report metrics
	t.Logf("Metrics for Go:")
	t.Logf("  True Positives: %d", metrics.TruePositives)
	t.Logf("  False Positives: %d", metrics.FalsePositives)
	t.Logf("  False Negatives: %d", metrics.FalseNegatives)
	t.Logf("  Precision: %.2f%%", metrics.Precision()*100)
	t.Logf("  Recall: %.2f%%", metrics.Recall()*100)
	t.Logf("  F1 Score: %.2f", metrics.F1Score())

	// Assert minimum quality thresholds
	assert.GreaterOrEqual(t, metrics.Precision(), 0.90, "Precision should be >= 90%")
	assert.GreaterOrEqual(t, metrics.Recall(), 0.90, "Recall should be >= 90%")
	assert.GreaterOrEqual(t, metrics.F1Score(), 0.90, "F1 Score should be >= 0.90")
}

// TestCallAnalysisMetrics_Java tests Java call analysis metrics
func TestCallAnalysisMetrics_Java(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	javaParser := parser.NewJavaParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "Metrics.java")
	content := `package com.example;

public class Metrics {
    public void method1() {
        method2();
        method3();
        System.out.println("test");
    }
    
    public void method2() {
        method3();
    }
    
    public void method3() {
        // leaf method
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := javaParser.Parse(file)
	require.NoError(t, err)

	// Define expected calls
	expectedCalls := []ExpectedCall{
		{"method1", "method2"},
		{"method1", "method3"},
		{"method1", "println"},
		{"method2", "method3"},
	}

	// Extract actual calls
	actualCalls := make(map[string]bool)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			key := fmt.Sprintf("%s->%s", dep.Source, dep.Target)
			actualCalls[key] = true
		}
	}

	// Calculate metrics
	metrics := CallAnalysisMetrics{}

	for _, expected := range expectedCalls {
		key := fmt.Sprintf("%s->%s", expected.Source, expected.Target)
		if actualCalls[key] {
			metrics.TruePositives++
			delete(actualCalls, key)
		} else {
			metrics.FalseNegatives++
			t.Logf("False Negative: %s -> %s", expected.Source, expected.Target)
		}
	}

	metrics.FalsePositives = len(actualCalls)
	for key := range actualCalls {
		t.Logf("False Positive: %s", key)
	}

	// Report metrics
	t.Logf("Metrics for Java:")
	t.Logf("  True Positives: %d", metrics.TruePositives)
	t.Logf("  False Positives: %d", metrics.FalsePositives)
	t.Logf("  False Negatives: %d", metrics.FalseNegatives)
	t.Logf("  Precision: %.2f%%", metrics.Precision()*100)
	t.Logf("  Recall: %.2f%%", metrics.Recall()*100)
	t.Logf("  F1 Score: %.2f", metrics.F1Score())

	// Assert minimum quality thresholds
	assert.GreaterOrEqual(t, metrics.Precision(), 0.85, "Precision should be >= 85%")
	assert.GreaterOrEqual(t, metrics.Recall(), 0.85, "Recall should be >= 85%")
}

// TestCallAnalysisMetrics_Python tests Python call analysis metrics
func TestCallAnalysisMetrics_Python(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	pythonParser := parser.NewPythonParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "metrics.py")
	content := `def function1():
    function2()
    function3()
    print("test")

def function2():
    function3()

def function3():
    pass
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "python",
	}

	parsedFile, err := pythonParser.Parse(file)
	require.NoError(t, err)

	// Define expected calls
	expectedCalls := []ExpectedCall{
		{"function1", "function2"},
		{"function1", "function3"},
		{"function1", "print"},
		{"function2", "function3"},
	}

	// Extract actual calls
	actualCalls := make(map[string]bool)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			key := fmt.Sprintf("%s->%s", dep.Source, dep.Target)
			actualCalls[key] = true
		}
	}

	// Calculate metrics
	metrics := CallAnalysisMetrics{}

	for _, expected := range expectedCalls {
		key := fmt.Sprintf("%s->%s", expected.Source, expected.Target)
		if actualCalls[key] {
			metrics.TruePositives++
			delete(actualCalls, key)
		} else {
			metrics.FalseNegatives++
			t.Logf("False Negative: %s -> %s", expected.Source, expected.Target)
		}
	}

	metrics.FalsePositives = len(actualCalls)
	for key := range actualCalls {
		t.Logf("False Positive: %s", key)
	}

	// Report metrics
	t.Logf("Metrics for Python:")
	t.Logf("  True Positives: %d", metrics.TruePositives)
	t.Logf("  False Positives: %d", metrics.FalsePositives)
	t.Logf("  False Negatives: %d", metrics.FalseNegatives)
	t.Logf("  Precision: %.2f%%", metrics.Precision()*100)
	t.Logf("  Recall: %.2f%%", metrics.Recall()*100)
	t.Logf("  F1 Score: %.2f", metrics.F1Score())

	// Assert minimum quality thresholds
	assert.GreaterOrEqual(t, metrics.Precision(), 0.85, "Precision should be >= 85%")
	assert.GreaterOrEqual(t, metrics.Recall(), 0.85, "Recall should be >= 85%")
}

// TestCallAnalysisMetrics_CrossLanguage_KotlinJava tests Kotlin-Java interop metrics
func TestCallAnalysisMetrics_CrossLanguage_KotlinJava(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	kotlinParser := parser.NewKotlinParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "KotlinJavaInterop.kt")
	content := `package com.example

import java.util.ArrayList
import java.util.HashMap

class KotlinJavaInterop {
    fun useJavaCollections() {
        val list = ArrayList<String>()
        list.add("item1")
        list.add("item2")
        
        val map = HashMap<String, Int>()
        map.put("key1", 1)
        map.put("key2", 2)
        
        val size = list.size
        val value = map.get("key1")
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := kotlinParser.Parse(file)
	require.NoError(t, err)

	// Define expected Java API calls
	expectedCalls := []ExpectedCall{
		{"useJavaCollections", "ArrayList"},
		{"useJavaCollections", "add"},
		{"useJavaCollections", "HashMap"},
		{"useJavaCollections", "put"},
		{"useJavaCollections", "get"},
	}

	// Extract actual calls
	actualCalls := make(map[string]bool)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			key := fmt.Sprintf("%s->%s", dep.Source, dep.Target)
			actualCalls[key] = true
		}
	}

	// Calculate metrics
	metrics := CallAnalysisMetrics{}

	for _, expected := range expectedCalls {
		key := fmt.Sprintf("%s->%s", expected.Source, expected.Target)
		if actualCalls[key] {
			metrics.TruePositives++
			delete(actualCalls, key)
		} else {
			metrics.FalseNegatives++
			t.Logf("False Negative: %s -> %s", expected.Source, expected.Target)
		}
	}

	metrics.FalsePositives = len(actualCalls)
	for key := range actualCalls {
		t.Logf("False Positive: %s", key)
	}

	// Report metrics
	t.Logf("Metrics for Kotlin-Java interop:")
	t.Logf("  True Positives: %d", metrics.TruePositives)
	t.Logf("  False Positives: %d", metrics.FalsePositives)
	t.Logf("  False Negatives: %d", metrics.FalseNegatives)
	t.Logf("  Precision: %.2f%%", metrics.Precision()*100)
	t.Logf("  Recall: %.2f%%", metrics.Recall()*100)
	t.Logf("  F1 Score: %.2f", metrics.F1Score())

	// Assert minimum quality thresholds for cross-language
	assert.GreaterOrEqual(t, metrics.Precision(), 0.80, "Precision should be >= 80% for cross-language")
	assert.GreaterOrEqual(t, metrics.Recall(), 0.70, "Recall should be >= 70% for cross-language")
}

// TestCallAnalysisMetrics_CrossLanguage_SwiftObjC tests Swift-ObjC interop metrics
func TestCallAnalysisMetrics_CrossLanguage_SwiftObjC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	swiftParser := parser.NewSwiftParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "SwiftObjCInterop.swift")
	content := `import Foundation

class SwiftObjCInterop {
    func useFoundationAPIs() {
        let str = NSString(string: "test")
        let length = str.length
        
        let array = NSArray(array: [1, 2, 3])
        let count = array.count
        
        let dict = NSDictionary(dictionary: ["key": "value"])
        let value = dict.object(forKey: "key")
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := swiftParser.Parse(file)
	require.NoError(t, err)

	// Define expected Objective-C API calls
	expectedCalls := []ExpectedCall{
		{"useFoundationAPIs", "NSString"},
		{"useFoundationAPIs", "NSArray"},
		{"useFoundationAPIs", "NSDictionary"},
		{"useFoundationAPIs", "object"},
	}

	// Extract actual calls
	actualCalls := make(map[string]bool)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			key := fmt.Sprintf("%s->%s", dep.Source, dep.Target)
			actualCalls[key] = true
		}
	}

	// Calculate metrics
	metrics := CallAnalysisMetrics{}

	for _, expected := range expectedCalls {
		key := fmt.Sprintf("%s->%s", expected.Source, expected.Target)
		if actualCalls[key] {
			metrics.TruePositives++
			delete(actualCalls, key)
		} else {
			metrics.FalseNegatives++
			t.Logf("False Negative: %s -> %s", expected.Source, expected.Target)
		}
	}

	metrics.FalsePositives = len(actualCalls)
	for key := range actualCalls {
		t.Logf("False Positive: %s", key)
	}

	// Report metrics
	t.Logf("Metrics for Swift-ObjC interop:")
	t.Logf("  True Positives: %d", metrics.TruePositives)
	t.Logf("  False Positives: %d", metrics.FalsePositives)
	t.Logf("  False Negatives: %d", metrics.FalseNegatives)
	t.Logf("  Precision: %.2f%%", metrics.Precision()*100)
	t.Logf("  Recall: %.2f%%", metrics.Recall()*100)
	t.Logf("  F1 Score: %.2f", metrics.F1Score())

	// Assert minimum quality thresholds for cross-language
	assert.GreaterOrEqual(t, metrics.Precision(), 0.75, "Precision should be >= 75% for cross-language")
	assert.GreaterOrEqual(t, metrics.Recall(), 0.65, "Recall should be >= 65% for cross-language")
}

// TestCallAnalysisMetrics_AllLanguages runs metrics tests for all supported languages
func TestCallAnalysisMetrics_AllLanguages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Run sub-tests for each language
	t.Run("Go", TestCallAnalysisMetrics_Go)
	t.Run("Java", TestCallAnalysisMetrics_Java)
	t.Run("Python", TestCallAnalysisMetrics_Python)
	t.Run("KotlinJava", TestCallAnalysisMetrics_CrossLanguage_KotlinJava)
	t.Run("SwiftObjC", TestCallAnalysisMetrics_CrossLanguage_SwiftObjC)
}
