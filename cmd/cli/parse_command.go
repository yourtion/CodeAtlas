package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

// ParseCommand represents the parse command configuration
type ParseCommand struct {
	Path          string
	Output        string
	File          string
	Language      string
	Workers       int
	Semantic      bool
	Verbose       bool
	IgnoreFile    string
	IgnorePattern []string
	NoIgnore      bool
}

// createParseCommand creates the parse CLI command
func createParseCommand() *cli.Command {
	return &cli.Command{
		Name:  "parse",
		Usage: "Parse source code and output structured JSON AST",
		Description: `Parse source code files and generate structured JSON output containing:
   - File metadata (path, language, size, checksum)
   - AST nodes (functions, classes, imports, etc.)
   - Symbols with signatures and docstrings
   - Dependency relationships between code entities

   Supports Go, JavaScript, TypeScript, and Python languages.

EXAMPLES:
   # Parse entire repository and output to stdout
   codeatlas parse --path /path/to/repo

   # Parse repository and save to file
   codeatlas parse --path /path/to/repo --output output.json

   # Parse single file
   codeatlas parse --file main.go

   # Parse with language filter
   codeatlas parse --path /path/to/repo --language go

   # Parse with custom ignore patterns
   codeatlas parse --path /path/to/repo --ignore-pattern "*.test.js" --ignore-pattern "*.spec.ts"

   # Parse with verbose output
   codeatlas parse --path /path/to/repo --verbose

   # Parse with semantic enhancement (requires CODEATLAS_LLM_API_KEY)
   codeatlas parse --path /path/to/repo --semantic

   # Parse with custom worker count
   codeatlas parse --path /path/to/repo --workers 8

ENVIRONMENT VARIABLES:
   CODEATLAS_LLM_API_KEY    API key for LLM-based semantic enhancement (optional)
   CODEATLAS_WORKERS        Default number of concurrent workers (default: number of CPUs)
   CODEATLAS_VERBOSE        Enable verbose logging (true/false)`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to repository or directory to parse",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Parse a single file instead of a directory",
			},
			&cli.StringFlag{
				Name:    "language",
				Aliases: []string{"l"},
				Usage:   "Filter files by language (go, javascript, typescript, python)",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Usage:   "Number of concurrent workers",
				Value:   runtime.NumCPU(),
			},
			&cli.BoolFlag{
				Name:  "semantic",
				Usage: "Enable LLM-based semantic enhancement (requires CODEATLAS_LLM_API_KEY)",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
			},
			&cli.StringFlag{
				Name:  "ignore-file",
				Usage: "Path to custom ignore file",
			},
			&cli.StringSliceFlag{
				Name:  "ignore-pattern",
				Usage: "Custom ignore pattern (can be specified multiple times)",
			},
			&cli.BoolFlag{
				Name:  "no-ignore",
				Usage: "Disable all ignore rules (parse all files)",
			},
		},
		Action: executeParseCommand,
	}
}

// executeParseCommand executes the parse command
func executeParseCommand(c *cli.Context) error {
	// Get workers from flag or environment variable
	workers := c.Int("workers")
	if workers == 0 {
		if envWorkers := os.Getenv("CODEATLAS_WORKERS"); envWorkers != "" {
			fmt.Sscanf(envWorkers, "%d", &workers)
		}
		if workers == 0 {
			workers = runtime.NumCPU()
		}
	}

	// Get verbose from flag or environment variable
	verbose := c.Bool("verbose")
	if !verbose {
		if envVerbose := os.Getenv("CODEATLAS_VERBOSE"); envVerbose == "true" || envVerbose == "1" {
			verbose = true
		}
	}

	// Get semantic flag - check for API key in environment
	semantic := c.Bool("semantic")
	if semantic {
		if os.Getenv("CODEATLAS_LLM_API_KEY") == "" {
			return fmt.Errorf("--semantic flag requires CODEATLAS_LLM_API_KEY environment variable to be set")
		}
	}

	cmd := ParseCommand{
		Path:          c.String("path"),
		Output:        c.String("output"),
		File:          c.String("file"),
		Language:      c.String("language"),
		Workers:       workers,
		Semantic:      semantic,
		Verbose:       verbose,
		IgnoreFile:    c.String("ignore-file"),
		IgnorePattern: c.StringSlice("ignore-pattern"),
		NoIgnore:      c.Bool("no-ignore"),
	}

	return cmd.Execute()
}

// Execute runs the parse command
func (cmd *ParseCommand) Execute() error {
	// Validate input: either --path or --file must be specified
	if cmd.Path == "" && cmd.File == "" {
		return fmt.Errorf("either --path or --file must be specified")
	}

	if cmd.Path != "" && cmd.File != "" {
		return fmt.Errorf("cannot specify both --path and --file")
	}

	// Create logger
	logger := utils.NewLogger(cmd.Verbose)

	// Determine if we're parsing a single file or a directory
	var files []parser.ScannedFile
	var err error

	if cmd.File != "" {
		// Single file mode
		files, err = cmd.scanSingleFile(logger)
	} else {
		// Directory mode
		files, err = cmd.scanDirectory(logger)
	}

	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	if len(files) == 0 {
		logger.Warn("No files found to parse")
		return nil
	}

	logger.Info("Found %d files to parse", len(files))

	// Initialize Tree-sitter parser
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		return fmt.Errorf("failed to initialize Tree-sitter parser: %w", err)
	}

	// Create parser pool
	pool := parser.NewParserPool(cmd.Workers, tsParser)
	pool.SetVerbose(cmd.Verbose)
	
	if cmd.Verbose {
		pool.SetProgressLogger(&parser.DefaultProgressLogger{})
	}

	logger.Info("Starting parsing with %d workers", cmd.Workers)
	startTime := time.Now()

	// Process files
	parsedFiles, parseErrors := pool.Process(files)

	parseTime := time.Since(startTime)
	logger.Info("Parsed %d files successfully, %d errors in %v", len(parsedFiles), len(parseErrors), parseTime)
	logger.Debug("Average time per file: %v", parseTime/time.Duration(len(files)))

	// Map to schema
	mapper := schema.NewSchemaMapper()
	var schemaFiles []schema.File
	var allEdges []schema.DependencyEdge
	var mappingErrors []schema.ParseError

	logger.Debug("Starting schema mapping for %d files", len(parsedFiles))
	mapStartTime := time.Now()

	for i, parsedFile := range parsedFiles {
		logger.Debug("[%d/%d] Mapping file: %s", i+1, len(parsedFiles), parsedFile.Path)
		
		schemaFile, edges, err := mapper.MapToSchema(parsedFile)
		if err != nil {
			mappingErrors = append(mappingErrors, schema.ParseError{
				File:    parsedFile.Path,
				Message: err.Error(),
				Type:    schema.ErrorMapping,
			})
			logger.Error("Failed to map file %s: %v", parsedFile.Path, err)
			continue
		}

		schemaFiles = append(schemaFiles, *schemaFile)
		allEdges = append(allEdges, edges...)
		
		logger.Debug("Mapped %d symbols and %d edges from %s", len(schemaFile.Symbols), len(edges), parsedFile.Path)
	}

	mapTime := time.Since(mapStartTime)
	logger.Debug("Schema mapping completed in %v", mapTime)

	// Collect all errors with detailed information
	var allErrors []schema.ParseError
	for _, err := range parseErrors {
		// Check if it's a DetailedParseError
		if detailedErr, ok := err.(*parser.DetailedParseError); ok {
			allErrors = append(allErrors, schema.ParseError{
				File:    detailedErr.File,
				Line:    detailedErr.Line,
				Column:  detailedErr.Column,
				Message: detailedErr.Message,
				Type:    schema.ErrorType(detailedErr.Type),
			})
		} else {
			// Generic error
			allErrors = append(allErrors, schema.ParseError{
				Message: err.Error(),
				Type:    schema.ErrorParse,
			})
		}
	}
	allErrors = append(allErrors, mappingErrors...)
	
	logger.Debug("Total errors collected: %d (%d parse errors, %d mapping errors)", 
		len(allErrors), len(parseErrors), len(mappingErrors))

	// Create output
	output := schema.ParseOutput{
		Files:         schemaFiles,
		Relationships: allEdges,
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   len(files),
			SuccessCount: len(schemaFiles),
			FailureCount: len(allErrors),
			Errors:       allErrors,
		},
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	// Write output
	if cmd.Output != "" {
		// Write to file
		err = os.WriteFile(cmd.Output, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		logger.Info("Output written to %s", cmd.Output)
	} else {
		// Write to stdout
		fmt.Println(string(jsonData))
	}

	// Print summary
	if cmd.Verbose {
		cmd.printSummary(output.Metadata, output)
	}

	// Log final statistics
	logger.Info("Parsing complete: %d/%d files successful", output.Metadata.SuccessCount, output.Metadata.TotalFiles)
	if len(allErrors) > 0 {
		logger.Warn("%d errors encountered during parsing", len(allErrors))
	}

	return nil
}

// scanSingleFile scans a single file
func (cmd *ParseCommand) scanSingleFile(logger *utils.Logger) ([]parser.ScannedFile, error) {
	// Check if file exists
	info, err := os.Stat(cmd.File)
	if err != nil {
		return nil, fmt.Errorf("file does not exist: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("specified path is a directory, use --path instead")
	}

	// Get absolute path
	absPath, err := filepath.Abs(cmd.File)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Detect language
	language := parser.DetermineLanguage(cmd.File)
	
	// Check if language is supported
	if language != "Go" && language != "JavaScript" && language != "TypeScript" && language != "Python" {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Apply language filter if specified
	if cmd.Language != "" {
		if language != cmd.Language {
			return nil, fmt.Errorf("file language %s does not match filter %s", language, cmd.Language)
		}
	}

	scannedFile := parser.ScannedFile{
		Path:     cmd.File,
		AbsPath:  absPath,
		Language: language,
		Size:     info.Size(),
	}

	logger.Info("Scanning single file: %s (language: %s)", cmd.File, language)

	return []parser.ScannedFile{scannedFile}, nil
}

// scanDirectory scans a directory for source files
func (cmd *ParseCommand) scanDirectory(logger *utils.Logger) ([]parser.ScannedFile, error) {
	// Check if directory exists
	if _, err := os.Stat(cmd.Path); err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	// Create ignore filter
	var filter *parser.IgnoreFilter
	var err error

	if !cmd.NoIgnore {
		// Collect .gitignore paths
		var gitignorePaths []string
		
		// Check for .gitignore in root
		gitignorePath := filepath.Join(cmd.Path, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			gitignorePaths = append(gitignorePaths, gitignorePath)
		}

		// Add custom ignore file if specified
		if cmd.IgnoreFile != "" {
			if _, err := os.Stat(cmd.IgnoreFile); err == nil {
				gitignorePaths = append(gitignorePaths, cmd.IgnoreFile)
			} else {
				logger.Warn("Ignore file not found: %s", cmd.IgnoreFile)
			}
		}

		// Create filter with custom patterns
		filter, err = parser.NewIgnoreFilter(gitignorePaths, cmd.IgnorePattern)
		if err != nil {
			return nil, fmt.Errorf("failed to create ignore filter: %w", err)
		}

		logger.Info("Ignore filter created with %d .gitignore files and %d custom patterns", 
			len(gitignorePaths), len(cmd.IgnorePattern))
	} else {
		logger.Info("Ignore rules disabled, parsing all files")
	}

	// Create file scanner
	scanner := parser.NewFileScanner(cmd.Path, filter)

	// Apply language filter if specified
	if cmd.Language != "" {
		scanner.SetLanguageFilter([]string{cmd.Language})
		logger.Info("Language filter applied: %s", cmd.Language)
	}

	// Scan directory
	logger.Info("Scanning directory: %s", cmd.Path)
	files, err := scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return files, nil
}

// printSummary prints a summary of the parsing operation
func (cmd *ParseCommand) printSummary(metadata schema.ParseMetadata, output schema.ParseOutput) {
	fmt.Println("\n=== Parse Summary ===")
	fmt.Printf("Version: %s\n", metadata.Version)
	fmt.Printf("Timestamp: %s\n", metadata.Timestamp.Format(time.RFC3339))
	fmt.Printf("\nFiles:\n")
	fmt.Printf("  Total files scanned: %d\n", metadata.TotalFiles)
	fmt.Printf("  Successfully parsed: %d\n", metadata.SuccessCount)
	fmt.Printf("  Failed: %d\n", metadata.FailureCount)
	fmt.Printf("  Success rate: %.1f%%\n", float64(metadata.SuccessCount)/float64(metadata.TotalFiles)*100)
	
	// Count symbols by type
	symbolCounts := make(map[schema.SymbolKind]int)
	totalSymbols := 0
	for _, file := range output.Files {
		for _, symbol := range file.Symbols {
			symbolCounts[symbol.Kind]++
			totalSymbols++
		}
	}
	
	if totalSymbols > 0 {
		fmt.Printf("\nSymbols extracted:\n")
		fmt.Printf("  Total: %d\n", totalSymbols)
		for kind, count := range symbolCounts {
			fmt.Printf("  %s: %d\n", kind, count)
		}
	}
	
	// Count relationships by type
	edgeCounts := make(map[schema.EdgeType]int)
	totalEdges := len(output.Relationships)
	for _, edge := range output.Relationships {
		edgeCounts[edge.EdgeType]++
	}
	
	if totalEdges > 0 {
		fmt.Printf("\nRelationships extracted:\n")
		fmt.Printf("  Total: %d\n", totalEdges)
		for edgeType, count := range edgeCounts {
			fmt.Printf("  %s: %d\n", edgeType, count)
		}
	}
	
	// Error breakdown
	if len(metadata.Errors) > 0 {
		errorCounts := make(map[schema.ErrorType]int)
		for _, err := range metadata.Errors {
			errorCounts[err.Type]++
		}
		
		fmt.Printf("\nError breakdown:\n")
		for errType, count := range errorCounts {
			fmt.Printf("  %s: %d\n", errType, count)
		}
		
		fmt.Println("\nError details (showing first 10):")
		for i, err := range metadata.Errors {
			if i >= 10 {
				fmt.Printf("... and %d more errors\n", len(metadata.Errors)-10)
				break
			}
			if err.Line > 0 {
				fmt.Printf("  - %s:%d:%d: %s (%s)\n", err.File, err.Line, err.Column, err.Message, err.Type)
			} else {
				fmt.Printf("  - %s: %s (%s)\n", err.File, err.Message, err.Type)
			}
		}
	}
}
