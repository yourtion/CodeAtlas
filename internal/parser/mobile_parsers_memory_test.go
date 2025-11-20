package parser

import (
"runtime"
"testing"
)

// TestMemoryUsageKotlinParser tests memory usage for Kotlin parser
func TestMemoryUsageKotlinParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	kotlinFile := createLargeKotlinFile(t, testDir, "large.kt")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		_, _ = parser.Parse(kotlinFile)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	t.Logf("Kotlin Parser - Total allocated: %.2f MB for 100 parses", allocatedMB)
	t.Logf("Kotlin Parser - Average per parse: %.2f KB", allocatedMB*1024/100)

	heapGrowthMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	if heapGrowthMB > 100 {
		t.Errorf("Excessive heap growth: %.2f MB (possible memory leak)", heapGrowthMB)
	}
}

// TestMemoryUsageJavaParser tests memory usage for Java parser
func TestMemoryUsageJavaParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	javaFile := createLargeJavaFile(t, testDir, "large.java")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		_, _ = parser.Parse(javaFile)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	t.Logf("Java Parser - Total allocated: %.2f MB for 100 parses", allocatedMB)
	t.Logf("Java Parser - Average per parse: %.2f KB", allocatedMB*1024/100)

	heapGrowthMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	if heapGrowthMB > 100 {
		t.Errorf("Excessive heap growth: %.2f MB (possible memory leak)", heapGrowthMB)
	}
}

// TestMemoryUsageSwiftParser tests memory usage for Swift parser
func TestMemoryUsageSwiftParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	swiftFile := createLargeSwiftFile(t, testDir, "large.swift")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewSwiftParser(tsParser)

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		_, _ = parser.Parse(swiftFile)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	t.Logf("Swift Parser - Total allocated: %.2f MB for 100 parses", allocatedMB)
	t.Logf("Swift Parser - Average per parse: %.2f KB", allocatedMB*1024/100)

	heapGrowthMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	if heapGrowthMB > 100 {
		t.Errorf("Excessive heap growth: %.2f MB (possible memory leak)", heapGrowthMB)
	}
}

// TestMemoryUsageCppParser tests memory usage for C++ parser
func TestMemoryUsageCppParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	cppFile := createLargeCppFile(t, testDir, "large.cpp")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewCppParser(tsParser)

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		_, _ = parser.Parse(cppFile)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	t.Logf("C++ Parser - Total allocated: %.2f MB for 100 parses", allocatedMB)
	t.Logf("C++ Parser - Average per parse: %.2f KB", allocatedMB*1024/100)

	heapGrowthMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	if heapGrowthMB > 100 {
		t.Errorf("Excessive heap growth: %.2f MB (possible memory leak)", heapGrowthMB)
	}
}

// TestMemoryUsageParserPool tests memory usage with parser pool
func TestMemoryUsageParserPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	files := createMobileBenchmarkFiles(t, testDir, 50)

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	pool := NewParserPool(runtime.NumCPU(), tsParser)
	for i := 0; i < 10; i++ {
		_, _ = pool.Process(files)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	t.Logf("Parser Pool - Total allocated: %.2f MB for 10 runs (50 files each)", allocatedMB)
	t.Logf("Parser Pool - Average per run: %.2f MB", allocatedMB/10)

	heapGrowthMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	if heapGrowthMB > 200 {
		t.Errorf("Excessive heap growth in parser pool: %.2f MB (possible memory leak)", heapGrowthMB)
	}
}
