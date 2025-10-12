package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// Repository represents a repository to be uploaded
type Repository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// File represents a file to be uploaded
type File struct {
	RepositoryID int    `json:"repository_id"`
	Path         string `json:"path"`
	Content      string `json:"content"`
	Language     string `json:"language"`
	Size         int64  `json:"size"`
}

const (
	// Version is the current version of the CLI
	Version = "1.0.0"
)

func main() {
	app := &cli.App{
		Name:    "codeatlas",
		Usage:   "CodeAtlas CLI tool for code repository analysis",
		Version: Version,
		Commands: []*cli.Command{
			createParseCommand(),
			{
				Name:  "upload",
				Usage: "Upload repository to CodeAtlas server",
				Description: `Upload a local repository to the CodeAtlas server for analysis.
   This command scans the repository, uploads file metadata and content
   to the server for knowledge graph construction.

EXAMPLES:
   # Upload repository with auto-detected name
   codeatlas upload --path /path/to/repo --server http://localhost:8080

   # Upload with custom repository name
   codeatlas upload --path /path/to/repo --server http://localhost:8080 --name my-project

ENVIRONMENT VARIABLES:
   CODEATLAS_SERVER    Default server URL (can be overridden with --server flag)`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "Path to the repository",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "server",
						Aliases:  []string{"s"},
						Usage:    "CodeAtlas server URL (can also use CODEATLAS_SERVER env var)",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "Repository name",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					repoPath := c.String("path")
					serverURL := c.String("server")
					repoName := c.String("name")

					// Get server URL from environment variable if not provided
					if serverURL == "" {
						serverURL = os.Getenv("CODEATLAS_SERVER")
						if serverURL == "" {
							return fmt.Errorf("server URL must be specified via --server flag or CODEATLAS_SERVER environment variable")
						}
					}

					// Use directory name as repository name if not provided
					if repoName == "" {
						repoName = filepath.Base(repoPath)
					}

					fmt.Printf("Uploading repository '%s' at %s to server %s\n", repoName, repoPath, serverURL)

					// Create repository on server
					repoID, err := createRepository(serverURL, repoName, "")
					if err != nil {
						return fmt.Errorf("failed to create repository: %w", err)
					}

					fmt.Printf("Created repository with ID: %d\n", repoID)

					// Scan repository files
					files, err := parser.ScanRepository(repoPath)
					if err != nil {
						return fmt.Errorf("failed to scan repository: %w", err)
					}

					fmt.Printf("Found %d files to upload\n", len(files))

					// Upload files to server
					for i, file := range files {
						err := uploadFile(serverURL, repoID, file)
						if err != nil {
							return fmt.Errorf("failed to upload file %s: %w", file.Path, err)
						}
						fmt.Printf("Uploaded file %d/%d: %s\n", i+1, len(files), file.Path)
					}

					fmt.Println("Repository upload completed successfully!")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// createRepository creates a repository on the server
func createRepository(serverURL, name, url string) (int, error) {
	repo := Repository{
		Name: name,
		URL:  url,
	}

	jsonData, err := json.Marshal(repo)
	if err != nil {
		return 0, err
	}

	resp, err := http.Post(serverURL+"/api/v1/repositories", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var createdRepo struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createdRepo); err != nil {
		return 0, err
	}

	return createdRepo.ID, nil
}

// uploadFile uploads a file to the server
func uploadFile(serverURL string, repoID int, fileInfo parser.FileInfo) error {
	file := File{
		RepositoryID: repoID,
		Path:         fileInfo.Path,
		Content:      fileInfo.Content,
		Language:     fileInfo.Language,
		Size:         fileInfo.Size,
	}

	jsonData, err := json.Marshal(file)
	if err != nil {
		return err
	}

	resp, err := http.Post(serverURL+"/api/v1/files", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}