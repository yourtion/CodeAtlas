package parser

import (
	"fmt"
	"runtime"
	"sync"
)

// ParserPool manages concurrent parsing with a worker pool pattern
type ParserPool struct {
	workers  int
	tsParser *TreeSitterParser
	verbose  bool
	logger   ProgressLogger
}

// ProgressLogger defines the interface for progress tracking
type ProgressLogger interface {
	LogProgress(current, total int, file string)
	LogError(file string, err error)
}

// ParseJob represents a file to be parsed
type ParseJob struct {
	File ScannedFile
}

// ParseResult represents the result of parsing a file
type ParseResult struct {
	File  *ParsedFile
	Error error
}

// NewParserPool creates a new parser pool with the specified number of workers
func NewParserPool(workers int, tsParser *TreeSitterParser) *ParserPool {
	// Default to number of CPUs if workers <= 0
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Optimize worker count based on CPU cores
	// For small file counts, fewer workers may be more efficient
	// Cap at 16 workers to avoid excessive context switching
	if workers > 16 {
		workers = 16
	}

	return &ParserPool{
		workers:  workers,
		tsParser: tsParser,
		verbose:  false,
	}
}

// OptimalWorkerCount returns the optimal number of workers for a given file count
func OptimalWorkerCount(fileCount int) int {
	cpus := runtime.NumCPU()

	// For very small file counts, use fewer workers
	if fileCount < 10 {
		return min(2, cpus)
	}

	// For small file counts, use half the CPUs
	if fileCount < 50 {
		return min(cpus/2, cpus)
	}

	// For medium to large file counts, use all CPUs but cap at 16
	return min(cpus, 16)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetVerbose enables or disables verbose logging
func (p *ParserPool) SetVerbose(verbose bool) {
	p.verbose = verbose
}

// SetProgressLogger sets a custom progress logger
func (p *ParserPool) SetProgressLogger(logger ProgressLogger) {
	p.logger = logger
}

// Process distributes files across workers and collects results
func (p *ParserPool) Process(files []ScannedFile) ([]*ParsedFile, []error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Create channels for job distribution and result collection
	jobs := make(chan ParseJob, len(files))
	results := make(chan ParseResult, len(files))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go p.worker(i, jobs, results, &wg)
	}

	// Send jobs to workers
	for _, file := range files {
		jobs <- ParseJob{File: file}
	}
	close(jobs)

	// Wait for all workers to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var parsedFiles []*ParsedFile
	var errors []error
	processed := 0
	total := len(files)

	for result := range results {
		processed++

		if result.Error != nil {
			errors = append(errors, result.Error)
			// Get file path for error logging
			filePath := "unknown"
			if result.File != nil {
				filePath = result.File.Path
			}
			if p.logger != nil {
				p.logger.LogError(filePath, result.Error)
			}

			// Even with errors, we might have partial results
			if result.File != nil {
				parsedFiles = append(parsedFiles, result.File)
			}
		} else if result.File != nil {
			parsedFiles = append(parsedFiles, result.File)
		}

		// Log progress if verbose
		if p.verbose && p.logger != nil {
			fileName := "unknown"
			if result.File != nil {
				fileName = result.File.Path
			}
			p.logger.LogProgress(processed, total, fileName)
		}
	}

	return parsedFiles, errors
}

// worker processes jobs from the jobs channel
func (p *ParserPool) worker(id int, jobs <-chan ParseJob, results chan<- ParseResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Each worker needs its own parser instance to avoid thread-safety issues
	// Tree-sitter parsers are not thread-safe
	workerTSParser, err := NewTreeSitterParser()
	if err != nil {
		// If we can't create a parser, we can't process any jobs
		for range jobs {
			results <- ParseResult{
				File:  nil,
				Error: fmt.Errorf("worker %d: failed to create parser: %w", id, err),
			}
		}
		return
	}

	goParser := NewGoParser(workerTSParser)
	jsParser := NewJSParser(workerTSParser)
	pyParser := NewPythonParser(workerTSParser)

	for job := range jobs {
		file := job.File

		// Select the appropriate parser based on language
		var parsedFile *ParsedFile
		var parseErr error

		switch file.Language {
		case "Go":
			parsedFile, parseErr = goParser.Parse(file)
		case "JavaScript", "TypeScript":
			parsedFile, parseErr = jsParser.Parse(file)
		case "Python":
			parsedFile, parseErr = pyParser.Parse(file)
		default:
			parseErr = fmt.Errorf("unsupported language: %s", file.Language)
		}

		// Send result
		result := ParseResult{
			File:  parsedFile,
			Error: parseErr,
		}
		results <- result
	}
}

// DefaultProgressLogger is a simple console-based progress logger
type DefaultProgressLogger struct {
	mu sync.Mutex
}

// LogProgress logs parsing progress
func (l *DefaultProgressLogger) LogProgress(current, total int, file string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("[%d/%d] Parsing: %s\n", current, total, file)
}

// LogError logs parsing errors
func (l *DefaultProgressLogger) LogError(file string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("Error parsing %s: %v\n", file, err)
}
