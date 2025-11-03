package main

import (
	"context"
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
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// createIndexCommand creates the index CLI command
func createIndexCommand() *cli.Command {
	return &cli.Command{
		Name:  "index",
		Usage: "Parse and index repository to CodeAtlas server",
		Description: `Parse a local repository and send the structured output to the CodeAtlas
   API server for indexing into the knowledge graph. This command combines
   parsing and indexing in a single operation.

   The indexer will:
   - Parse source code files and extract symbols, AST nodes, and relationships
   - Send the parsed data to the API server
   - Store data in PostgreSQL, build AGE graph relationships
   - Generate vector embeddings for semantic search (unless --skip-vectors is set)

EXAMPLES:
   # Index repository with auto-detected name
   codeatlas index --path /path/to/repo --api-url http://localhost:8080

   # Index with custom repository metadata
   codeatlas index --path /path/to/repo --name my-project --url https://github.com/user/repo --branch main

   # Index incrementally (only changed files)
   codeatlas index --path /path/to/repo --incremental

   # Index without generating embeddings (faster)
   codeatlas index --path /path/to/repo --skip-vectors

   # Index with custom batch size and workers
   codeatlas index --path /path/to/repo --batch-size 50 --workers 8

   # Index from pre-parsed JSON output
   codeatlas index --input parsed-output.json --name my-project

ENVIRONMENT VARIABLES:
   CODEATLAS_API_URL        Default API server URL
   CODEATLAS_API_TOKEN      API authentication token
   CODEATLAS_WORKERS        Default number of concurrent workers`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Path to repository to parse and index",
			},
			&cli.StringFlag{
				Name:    "input",
				Aliases: []string{"i"},
				Usage:   "Path to pre-parsed JSON output file (alternative to --path)",
			},
			&cli.StringFlag{
				Name:    "repo-id",
				Aliases: []string{"r"},
				Usage:   "Repository ID (auto-generated if not provided)",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Repository name (defaults to directory name)",
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "Repository URL",
			},
			&cli.StringFlag{
				Name:    "branch",
				Aliases: []string{"b"},
				Usage:   "Repository branch",
				Value:   "main",
			},
			&cli.StringFlag{
				Name:  "commit",
				Usage: "Commit hash",
			},
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "API server URL (can also use CODEATLAS_API_URL env var)",
			},
			&cli.StringFlag{
				Name:  "api-token",
				Usage: "API authentication token (can also use CODEATLAS_API_TOKEN env var)",
			},
			&cli.BoolFlag{
				Name:  "incremental",
				Usage: "Only process changed files (based on checksums)",
			},
			&cli.BoolFlag{
				Name:  "skip-vectors",
				Usage: "Skip embedding generation (faster indexing)",
			},
			&cli.IntFlag{
				Name:  "batch-size",
				Usage: "Batch size for processing",
				Value: 100,
			},
			&cli.IntFlag{
				Name:  "workers",
				Usage: "Number of concurrent workers for parsing",
				Value: runtime.NumCPU(),
			},
			&cli.StringFlag{
				Name:  "embedding-model",
				Usage: "Embedding model to use",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 10 * time.Minute,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
			},
		},
		Action: executeIndexCommand,
	}
}

// executeIndexCommand executes the index command
func executeIndexCommand(c *cli.Context) error {
	// Validate input: either --path or --input must be specified
	path := c.String("path")
	inputFile := c.String("input")

	if path == "" && inputFile == "" {
		return fmt.Errorf("either --path or --input must be specified")
	}

	if path != "" && inputFile != "" {
		return fmt.Errorf("cannot specify both --path and --input")
	}

	// Get API URL from flag or environment
	apiURL := c.String("api-url")
	if apiURL == "" {
		apiURL = os.Getenv("CODEATLAS_API_URL")
		if apiURL == "" {
			return fmt.Errorf("API URL must be specified via --api-url flag or CODEATLAS_API_URL environment variable")
		}
	}

	// Get API token from flag or environment
	apiToken := c.String("api-token")
	if apiToken == "" {
		apiToken = os.Getenv("CODEATLAS_API_TOKEN")
	}

	// Create logger
	verbose := c.Bool("verbose")
	logger := utils.NewLogger(verbose)

	// Get repository name
	repoName := c.String("name")
	if repoName == "" {
		if path != "" {
			repoName = filepath.Base(path)
		} else {
			return fmt.Errorf("--name must be specified when using --input")
		}
	}

	logger.Info("Starting index operation for repository: %s", repoName)

	// Get or parse the output
	var parseOutput schema.ParseOutput
	var err error

	if inputFile != "" {
		// Load from pre-parsed JSON file
		logger.Info("Loading parse output from: %s", inputFile)
		parseOutput, err = loadParseOutput(inputFile)
		if err != nil {
			return fmt.Errorf("failed to load parse output: %w", err)
		}
		logger.Info("Loaded %d files from parse output", len(parseOutput.Files))
	} else {
		// Parse the repository
		logger.Info("Parsing repository at: %s", path)
		parseOutput, err = parseRepository(path, c.Int("workers"), verbose, logger)
		if err != nil {
			return fmt.Errorf("failed to parse repository: %w", err)
		}
		logger.Info("Parsed %d files successfully", parseOutput.Metadata.SuccessCount)
	}

	// Create API client
	clientOpts := []client.ClientOption{
		client.WithTimeout(c.Duration("timeout")),
		client.WithMaxRetries(3),
	}
	if apiToken != "" {
		clientOpts = append(clientOpts, client.WithToken(apiToken))
	}

	apiClient := client.NewAPIClient(apiURL, clientOpts...)

	// Check API health
	logger.Info("Checking API server health...")
	ctx := context.Background()
	if err := apiClient.Health(ctx); err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}
	logger.Info("API server is healthy")

	// Create index request
	indexReq := &client.IndexRequest{
		RepoID:      c.String("repo-id"),
		RepoName:    repoName,
		RepoURL:     c.String("url"),
		Branch:      c.String("branch"),
		CommitHash:  c.String("commit"),
		ParseOutput: parseOutput,
		Options: client.IndexOptions{
			Incremental:    c.Bool("incremental"),
			SkipVectors:    c.Bool("skip-vectors"),
			BatchSize:      c.Int("batch-size"),
			WorkerCount:    c.Int("workers"),
			EmbeddingModel: c.String("embedding-model"),
		},
	}

	// Send index request
	logger.Info("Sending index request to API server...")
	startTime := time.Now()

	indexResp, err := apiClient.Index(ctx, indexReq)
	if err != nil {
		return fmt.Errorf("index request failed: %w", err)
	}

	duration := time.Since(startTime)

	// Display results
	displayIndexResults(indexResp, duration, logger)

	// Return error if there were indexing errors
	if len(indexResp.Errors) > 0 {
		logger.Warn("Indexing completed with %d errors", len(indexResp.Errors))
		return cli.Exit("", 1)
	}

	logger.Info("Indexing completed successfully!")
	return nil
}

// parseRepository parses a repository and returns the parse output
func parseRepository(path string, workers int, verbose bool, logger *utils.Logger) (schema.ParseOutput, error) {
	// Check if directory exists
	if _, err := os.Stat(path); err != nil {
		return schema.ParseOutput{}, fmt.Errorf("path does not exist: %w", err)
	}

	// Create ignore filter
	var gitignorePaths []string
	gitignorePath := filepath.Join(path, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gitignorePaths = append(gitignorePaths, gitignorePath)
	}

	filter, err := parser.NewIgnoreFilter(gitignorePaths, nil)
	if err != nil {
		return schema.ParseOutput{}, fmt.Errorf("failed to create ignore filter: %w", err)
	}

	// Scan directory
	scanner := parser.NewFileScanner(path, filter)
	files, err := scanner.Scan()
	if err != nil {
		return schema.ParseOutput{}, fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		return schema.ParseOutput{}, fmt.Errorf("no files found to parse")
	}

	logger.Info("Found %d files to parse", len(files))

	// Initialize Tree-sitter parser
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		return schema.ParseOutput{}, fmt.Errorf("failed to initialize Tree-sitter parser: %w", err)
	}

	// Optimize worker count for small file sets
	if workers == runtime.NumCPU() && len(files) < 50 {
		workers = parser.OptimalWorkerCount(len(files))
		logger.Debug("Optimized worker count to %d for %d files", workers, len(files))
	}

	// Create parser pool
	pool := parser.NewParserPool(workers, tsParser)
	pool.SetVerbose(verbose)

	if verbose {
		pool.SetProgressLogger(&parser.DefaultProgressLogger{})
	}

	logger.Info("Parsing with %d workers", workers)
	startTime := time.Now()

	// Process files
	parsedFiles, parseErrors := pool.Process(files)

	parseTime := time.Since(startTime)
	logger.Info("Parsed %d files in %v", len(parsedFiles), parseTime)

	// Map to schema
	mapper := schema.NewSchemaMapper()
	var schemaFiles []schema.File
	var allEdges []schema.DependencyEdge
	var mappingErrors []schema.ParseError

	for _, parsedFile := range parsedFiles {
		schemaFile, edges, err := mapper.MapToSchema(parsedFile)
		if err != nil {
			mappingErrors = append(mappingErrors, schema.ParseError{
				File:    parsedFile.Path,
				Message: err.Error(),
				Type:    schema.ErrorMapping,
			})
			continue
		}

		schemaFiles = append(schemaFiles, *schemaFile)
		allEdges = append(allEdges, edges...)
	}

	// Collect all errors
	var allErrors []schema.ParseError
	for _, err := range parseErrors {
		if detailedErr, ok := err.(*parser.DetailedParseError); ok {
			allErrors = append(allErrors, schema.ParseError{
				File:    detailedErr.File,
				Line:    detailedErr.Line,
				Column:  detailedErr.Column,
				Message: detailedErr.Message,
				Type:    schema.ErrorType(detailedErr.Type),
			})
		} else {
			allErrors = append(allErrors, schema.ParseError{
				Message: err.Error(),
				Type:    schema.ErrorParse,
			})
		}
	}
	allErrors = append(allErrors, mappingErrors...)

	// Create output
	// FailureCount should be the number of files that failed, not the number of errors
	failedFiles := len(files) - len(schemaFiles)
	output := schema.ParseOutput{
		Files:         schemaFiles,
		Relationships: allEdges,
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   len(files),
			SuccessCount: len(schemaFiles),
			FailureCount: failedFiles,
			Errors:       allErrors,
		},
	}

	return output, nil
}

// loadParseOutput loads parse output from a JSON file
func loadParseOutput(filePath string) (schema.ParseOutput, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return schema.ParseOutput{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var output schema.ParseOutput
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&output); err != nil {
		return schema.ParseOutput{}, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return output, nil
}

// displayIndexResults displays the indexing results
func displayIndexResults(resp *client.IndexResponse, duration time.Duration, logger *utils.Logger) {
	fmt.Println("\n=== Index Results ===")
	fmt.Printf("Repository ID: %s\n", resp.RepoID)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Println()
	fmt.Printf("Files processed: %d\n", resp.FilesProcessed)
	fmt.Printf("Symbols created: %d\n", resp.SymbolsCreated)
	fmt.Printf("Edges created: %d\n", resp.EdgesCreated)
	fmt.Printf("Vectors created: %d\n", resp.VectorsCreated)

	if len(resp.Errors) > 0 {
		fmt.Printf("\nErrors encountered: %d\n", len(resp.Errors))
		fmt.Println("\nError details (showing first 10):")
		for i, err := range resp.Errors {
			if i >= 10 {
				fmt.Printf("... and %d more errors\n", len(resp.Errors)-10)
				break
			}
			if err.FilePath != "" {
				fmt.Printf("  - [%s] %s: %s\n", err.Type, err.FilePath, err.Message)
			} else {
				fmt.Printf("  - [%s] %s\n", err.Type, err.Message)
			}
			if err.Retryable {
				fmt.Printf("    (retryable)\n")
			}
		}
	}
}
