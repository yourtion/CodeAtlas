package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// createAskCommand creates the ask CLI command.
//
// Performs a QA context query against the API server and outputs a Markdown
// prompt ready to paste into an LLM. The prompt includes the relevant symbols
// together with their 1-hop callers/callees, optionally with inlined source.
func createAskCommand() *cli.Command {
	return &cli.Command{
		Name:  "ask",
		Usage: "Ask a question and get assembled code context (prompt for LLMs)",
		Description: `Performs a QA context query and outputs a Markdown prompt ready to paste into an LLM.
The prompt includes relevant symbols with their 1-hop callers/callees.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "question",
				Aliases:  []string{"q"},
				Usage:    "Natural language question",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "Filter by repository ID (repeatable for multiple repos)",
			},
			&cli.StringFlag{Name: "language", Aliases: []string{"l"}, Usage: "Filter by language"},
			&cli.StringFlag{Name: "kind", Aliases: []string{"k"}, Usage: "Filter by symbol kind (comma-separated)"},
			&cli.StringFlag{Name: "mode", Usage: "Retrieval mode: hybrid(default)|vector|keyword", Value: "hybrid"},
			&cli.IntFlag{Name: "limit", Usage: "Top-K results", Value: 10},
			&cli.BoolFlag{Name: "include-source", Usage: "Inline source code into prompt"},
			&cli.StringFlag{Name: "api-url", Usage: "API server URL (or CODEATLAS_API_URL env)"},
			&cli.StringFlag{Name: "api-token", Usage: "API auth token (or CODEATLAS_API_TOKEN env)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Write prompt to file (default stdout)"},
			&cli.BoolFlag{Name: "json", Usage: "Output full JSON response instead of prompt only"},
			&cli.DurationFlag{Name: "timeout", Usage: "Request timeout", Value: 60 * time.Second},
		},
		Action: executeAskCommand,
	}
}

// executeAskCommand runs the ask command.
func executeAskCommand(c *cli.Context) error {
	apiURL := c.String("api-url")
	if apiURL == "" {
		apiURL = os.Getenv("CODEATLAS_API_URL")
		if apiURL == "" {
			return fmt.Errorf("API URL required via --api-url or CODEATLAS_API_URL")
		}
	}
	apiToken := c.String("api-token")
	if apiToken == "" {
		apiToken = os.Getenv("CODEATLAS_API_TOKEN")
	}

	apiClient := client.NewAPIClient(apiURL, client.WithTimeout(c.Duration("timeout")), client.WithToken(apiToken))

	req := &client.QARequest{
		Query:         c.String("question"),
		RepoIDs:       c.StringSlice("repo"),
		Language:      c.String("language"),
		Mode:          c.String("mode"),
		Limit:         c.Int("limit"),
		IncludeSource: c.Bool("include-source"),
	}
	if k := c.String("kind"); k != "" {
		req.Kind = strings.Split(k, ",")
		// Trim whitespace from each kind
		for i := range req.Kind {
			req.Kind[i] = strings.TrimSpace(req.Kind[i])
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Duration("timeout"))
	defer cancel()

	resp, err := apiClient.Ask(ctx, req)
	if err != nil {
		return fmt.Errorf("ask failed: %w", err)
	}

	if c.Bool("json") {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON response: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	output := c.String("output")
	if output != "" {
		return os.WriteFile(output, []byte(resp.Prompt), 0644)
	}
	fmt.Print(resp.Prompt)
	return nil
}
