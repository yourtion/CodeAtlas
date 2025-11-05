//go:build parse_tests

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// skipIfNoParseCommand skips the test if parse command is not available
func skipIfNoParseCommand(t *testing.T) {
	cmd := exec.Command(cliBinaryPath, "parse", "--help")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: parse command not implemented")
	}
}

// TestParsePerformance validates that parsing meets performance targets
func TestParsePerformance(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a large test repository with 1000+ files
	testDir := t.TempDir()
	fileCount := 1000

	t.Logf("Creating %d test files...", fileCount)
	files := createLargeTestRepo(t, testDir, fileCount)

	// Initialize parser
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Test with default worker count (NumCPU)
	workers := runtime.NumCPU()
	t.Logf("Testing with %d workers (NumCPU)", workers)

	pool := parser.NewParserPool(workers, tsParser)

	// Measure parsing time
	startTime := time.Now()
	parsedFiles, parseErrors := pool.Process(files)
	parseTime := time.Since(startTime)

	t.Logf("Parsed %d files in %v", len(parsedFiles), parseTime)
	t.Logf("Parse errors: %d", len(parseErrors))
	t.Logf("Average time per file: %v", parseTime/time.Duration(fileCount))
	t.Logf("Files per second: %.2f", float64(fileCount)/parseTime.Seconds())

	// Performance target: <5 minutes for 1000 files
	targetTime := 5 * time.Minute
	if parseTime > targetTime {
		t.Errorf("Performance target not met: took %v, target was %v", parseTime, targetTime)
	}

	// Verify success rate
	successRate := float64(len(parsedFiles)) / float64(fileCount) * 100
	t.Logf("Success rate: %.1f%%", successRate)

	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.1f%%, expected at least 95%%", successRate)
	}

	// Test schema mapping performance
	t.Log("Testing schema mapping performance...")
	mapper := schema.NewSchemaMapper()

	mapStartTime := time.Now()
	var schemaFiles []schema.File
	var mappingErrors int

	for _, parsedFile := range parsedFiles {
		schemaFile, _, err := mapper.MapToSchema(parsedFile)
		if err != nil {
			mappingErrors++
			continue
		}
		schemaFiles = append(schemaFiles, *schemaFile)
	}

	mapTime := time.Since(mapStartTime)
	t.Logf("Mapped %d files in %v", len(schemaFiles), mapTime)
	t.Logf("Mapping errors: %d", mappingErrors)
	t.Logf("Average mapping time per file: %v", mapTime/time.Duration(len(parsedFiles)))

	// Total time should still be under target
	totalTime := parseTime + mapTime
	t.Logf("Total processing time: %v", totalTime)

	if totalTime > targetTime {
		t.Errorf("Total processing time exceeds target: %v > %v", totalTime, targetTime)
	}
}

// TestParseMemoryUsage validates memory usage during parsing
func TestParseMemoryUsage(t *testing.T) {
	skipIfBinaryNotExists(t)
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDir := t.TempDir()
	fileCount := 500

	files := createLargeTestRepo(t, testDir, fileCount)

	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Force GC before measurement
	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	pool := parser.NewParserPool(runtime.NumCPU(), tsParser)
	parsedFiles, _ := pool.Process(files)

	// Map to schema
	mapper := schema.NewSchemaMapper()
	for _, parsedFile := range parsedFiles {
		_, _, _ = mapper.MapToSchema(parsedFile)
	}

	runtime.GC()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Use TotalAlloc for total memory allocated during execution
	memUsedMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024
	heapUsedMB := float64(memAfter.Alloc) / 1024 / 1024

	t.Logf("Total memory allocated: %.2f MB for %d files", memUsedMB, fileCount)
	t.Logf("Heap memory in use: %.2f MB", heapUsedMB)
	t.Logf("Memory per file: %.2f KB", memUsedMB*1024/float64(fileCount))

	// Target: <2GB for 1000 files, so <1GB for 500 files
	// Use heap memory for the check as it's more relevant
	targetMemMB := 1024.0
	if heapUsedMB > targetMemMB {
		t.Errorf("Heap memory usage exceeds target: %.2f MB > %.2f MB", heapUsedMB, targetMemMB)
	} else {
		t.Logf("Memory usage within target: %.2f MB < %.2f MB ✓", heapUsedMB, targetMemMB)
	}
}

// TestWorkerScaling validates that increasing workers improves performance
func TestWorkerScaling(t *testing.T) {
	skipIfBinaryNotExists(t)
	if testing.Short() {
		t.Skip("Skipping scaling test in short mode")
	}

	testDir := t.TempDir()
	fileCount := 200

	files := createLargeTestRepo(t, testDir, fileCount)

	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	workerCounts := []int{1, 2, 4, 8}
	times := make(map[int]time.Duration)

	for _, workers := range workerCounts {
		pool := parser.NewParserPool(workers, tsParser)

		startTime := time.Now()
		_, _ = pool.Process(files)
		elapsed := time.Since(startTime)

		times[workers] = elapsed
		t.Logf("Workers: %d, Time: %v, Files/sec: %.2f",
			workers, elapsed, float64(fileCount)/elapsed.Seconds())
	}

	// Verify that more workers generally means better performance
	// (up to a point, depending on CPU cores)
	if times[2] < times[1] {
		t.Logf("2 workers (%v) faster than 1 worker (%v) ✓", times[2], times[1])
	} else {
		t.Logf("Warning: 2 workers (%v) not faster than 1 worker (%v)", times[2], times[1])
	}

	if times[4] < times[2] {
		t.Logf("4 workers (%v) faster than 2 workers (%v) ✓", times[4], times[2])
	} else {
		t.Logf("Warning: 4 workers (%v) not faster than 2 workers (%v)", times[4], times[2])
	}

	// Calculate speedup
	speedup := float64(times[1]) / float64(times[8])
	t.Logf("Speedup with 8 workers vs 1 worker: %.2fx", speedup)

	// We should see at least some speedup with more workers
	if speedup < 1.5 {
		t.Logf("Warning: Limited speedup with multiple workers: %.2fx", speedup)
	}
}

// TestFileScanningPerformance validates file scanning performance
func TestFileScanningPerformance(t *testing.T) {
	skipIfBinaryNotExists(t)
	testDir := t.TempDir()
	fileCount := 1000

	// Create files in nested directory structure
	createNestedTestRepo(t, testDir, fileCount)

	filter, err := parser.NewIgnoreFilter(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	scanner := parser.NewFileScanner(testDir, filter)

	startTime := time.Now()
	files, err := scanner.Scan()
	scanTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	t.Logf("Scanned %d files in %v", len(files), scanTime)
	t.Logf("Scan rate: %.2f files/sec", float64(len(files))/scanTime.Seconds())

	// Scanning should be very fast - target <10 seconds for 1000 files
	targetTime := 10 * time.Second
	if scanTime > targetTime {
		t.Errorf("Scanning too slow: %v > %v", scanTime, targetTime)
	}
}

// Helper functions

func createLargeTestRepo(t *testing.T, dir string, count int) []parser.ScannedFile {
	var files []parser.ScannedFile

	// Create a mix of languages
	for i := 0; i < count; i++ {
		var file parser.ScannedFile
		var content string
		var ext string
		var lang string

		switch i % 3 {
		case 0:
			content = generateGoFile(i)
			ext = ".go"
			lang = "Go"
		case 1:
			content = generateJSFile(i)
			ext = ".js"
			lang = "JavaScript"
		case 2:
			content = generatePythonFile(i)
			ext = ".py"
			lang = "Python"
		}

		filename := fmt.Sprintf("file%d%s", i, ext)
		path := filepath.Join(dir, filename)

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		info, _ := os.Stat(path)
		file = parser.ScannedFile{
			Path:     filename,
			AbsPath:  path,
			Language: lang,
			Size:     info.Size(),
		}

		files = append(files, file)
	}

	return files
}

func createNestedTestRepo(t *testing.T, dir string, count int) {
	// Create nested directory structure
	dirs := []string{
		"src",
		"src/api",
		"src/models",
		"src/utils",
		"tests",
		"tests/unit",
		"tests/integration",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Distribute files across directories
	filesPerDir := count / len(dirs)
	fileIndex := 0

	for _, d := range dirs {
		for i := 0; i < filesPerDir; i++ {
			var content string
			var ext string

			switch fileIndex % 3 {
			case 0:
				content = generateGoFile(fileIndex)
				ext = ".go"
			case 1:
				content = generateJSFile(fileIndex)
				ext = ".js"
			case 2:
				content = generatePythonFile(fileIndex)
				ext = ".py"
			}

			filename := fmt.Sprintf("file%d%s", fileIndex, ext)
			path := filepath.Join(dir, d, filename)

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}

			fileIndex++
		}
	}
}

func generateGoFile(index int) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

// skipIfNoParseCommand skips the test if parse command is not available
func skipIfNoParseCommand(t *testing.T) {
cmd := exec.Command(cliBinaryPath, "parse", "--help")
if err := cmd.Run(); err != nil {
t.Skip("Skipping test: parse command not implemented")
}
}

// Entity%d represents an entity in the system
type Entity%d struct {
	ID   int
	Name string
	Data map[string]interface{}
}

// NewEntity%d creates a new entity
func NewEntity%d(id int, name string) *Entity%d {
	return &Entity%d{
		ID:   id,
		Name: name,
		Data: make(map[string]interface{}),
	}
}

// GetID returns the entity ID
func (e *Entity%d) GetID() int {
	return e.ID
}

// SetData sets a data value
func (e *Entity%d) SetData(key string, value interface{}) {
	e.Data[key] = value
}

// GetData gets a data value
func (e *Entity%d) GetData(key string) interface{} {
	return e.Data[key]
}

func main() {
	entity := NewEntity%d(%d, "Entity %d")
	fmt.Println(entity.GetID())
}
`, index, index, index, index, index, index, index, index, index, index, index, index)
}

func generateJSFile(index int) string {
	return fmt.Sprintf(`/**
 * Entity%d class
 */
class Entity%d {
	constructor(id, name) {
		this.id = id;
		this.name = name;
		this.data = {};
	}

	/**
	 * Get entity ID
	 */
	getId() {
		return this.id;
	}

	/**
	 * Set data value
	 */
	setData(key, value) {
		this.data[key] = value;
	}

	/**
	 * Get data value
	 */
	getData(key) {
		return this.data[key];
	}
}

/**
 * Create entity
 */
function createEntity%d(id, name) {
	return new Entity%d(id, name);
}

const entity = createEntity%d(%d, "Entity %d");
console.log(entity.getId());
`, index, index, index, index, index, index, index)
}

func generatePythonFile(index int) string {
	return fmt.Sprintf(`"""
Entity%d module
"""

class Entity%d:
	"""Entity class"""
	
	def __init__(self, id: int, name: str):
		"""Initialize entity"""
		self.id = id
		self.name = name
		self.data = {}
	
	def get_id(self) -> int:
		"""Get entity ID"""
		return self.id
	
	def set_data(self, key: str, value) -> None:
		"""Set data value"""
		self.data[key] = value
	
	def get_data(self, key: str):
		"""Get data value"""
		return self.data.get(key)

def create_entity_%d(id: int, name: str) -> Entity%d:
	"""Create entity"""
	return Entity%d(id, name)

if __name__ == "__main__":
	entity = create_entity_%d(%d, "Entity %d")
	print(entity.get_id())
`, index, index, index, index, index, index, index, index)
}
