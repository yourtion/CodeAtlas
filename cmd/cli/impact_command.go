package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// createImpactCommand 创建影响范围可视化命令。
//
// 基于多跳可达性查询，把"传递调用链"或"反向影响范围"以分层缩进树形式输出，
// 直观展示改动一个符号会波及哪些代码（或一个符号的执行会触及哪些代码）。
func createImpactCommand() *cli.Command {
	return &cli.Command{
		Name:  "impact",
		Usage: "Visualize the impact radius of a symbol (multi-hop call chain)",
		Description: `Show all symbols reachable from a starting symbol along call edges,
displayed as a depth-grouped indented tree.

Two directions are supported:
  - callees (default): "what code does this symbol's execution touch"
  - callers           : "what code is affected if this symbol changes"

This is a pure relationship query and does NOT generate embeddings.

EXAMPLES:
   # Show transitive callees of a symbol (what it calls, directly and indirectly)
   codeatlas impact --symbol abc-123

   # Show reverse impact (who calls it, directly and indirectly)
   codeatlas impact --symbol abc-123 --direction callers

   # Limit traversal depth
   codeatlas impact --symbol abc-123 --depth 3

ENVIRONMENT VARIABLES:
   CODEATLAS_API_URL        Default API server URL
   CODEATLAS_API_TOKEN      API authentication token`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "symbol",
				Aliases:  []string{"s"},
				Usage:    "Starting symbol ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "direction",
				Aliases: []string{"d"},
				Usage:   "Traversal direction: callees (forward) or callers (backward)",
				Value:   "callees",
			},
			&cli.IntFlag{
				Name:    "depth",
				Usage:   "Maximum hop count (default uses server default, typically 5)",
				Value:   0,
			},
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "API server URL (can also use CODEATLAS_API_URL env var)",
			},
			&cli.StringFlag{
				Name:  "api-token",
				Usage: "API authentication token (can also use CODEATLAS_API_TOKEN env var)",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 30 * time.Second,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
			},
		},
		Action: executeImpactCommand,
	}
}

// executeImpactCommand 执行 impact 命令。
func executeImpactCommand(c *cli.Context) error {
	symbolID := c.String("symbol")
	if symbolID == "" {
		return fmt.Errorf("symbol ID is required (--symbol)")
	}

	direction := c.String("direction")
	if direction != "callees" && direction != "callers" {
		return fmt.Errorf("direction must be 'callees' or 'callers', got: %s", direction)
	}

	// Get API URL from flag or environment
	apiURL := c.String("api-url")
	if apiURL == "" {
		apiURL = os.Getenv("CODEATLAS_API_URL")
		if apiURL == "" {
			return fmt.Errorf("API URL must be specified via --api-url flag or CODEATLAS_API_URL environment variable")
		}
	}

	apiToken := c.String("api-token")
	if apiToken == "" {
		apiToken = os.Getenv("CODEATLAS_API_TOKEN")
	}

	verbose := c.Bool("verbose")
	logger := utils.NewLogger(verbose)

	// Create API client
	clientOpts := []client.ClientOption{
		client.WithTimeout(c.Duration("timeout")),
		client.WithMaxRetries(3),
	}
	if apiToken != "" {
		clientOpts = append(clientOpts, client.WithToken(apiToken))
	}
	apiClient := client.NewAPIClient(apiURL, clientOpts...)

	ctx := context.Background()

	// Check API health
	logger.Info("Checking API server health...")
	if err := apiClient.Health(ctx); err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}

	depth := c.Int("depth")

	// Query transitive relationship
	label := "callees"
	if direction == "callers" {
		label = "callers"
	}
	logger.Info("Querying transitive %s for symbol %s (depth=%d)...", label, symbolID, depth)

	startTime := time.Now()
	var resp *client.TransitiveResponse
	var err error
	if direction == "callers" {
		resp, err = apiClient.GetTransitiveCallers(ctx, symbolID, depth)
	} else {
		resp, err = apiClient.GetTransitiveCallees(ctx, symbolID, depth)
	}
	if err != nil {
		return fmt.Errorf("failed to query transitive %s: %w", label, err)
	}
	duration := time.Since(startTime)

	// Render
	displayImpactTree(os.Stdout, resp, direction, duration)
	return nil
}

// displayImpactTree 把多跳可达集合按 depth 分组，以缩进树形式写入 w。
//
// 注：多跳 API 返回的是去重可达集合（每符号最短跳数），不是完整路径。
// 因此这里按 depth 分层缩进展示（同层符号按名字排序），形如：
//
//	callees (depth=1):
//	├── processData [function] main.go
//	└── validateInput [function] main.go
//	callees (depth=2):
//	├── saveToDB [function] db.go
//	└── logEvent [function] log.go
//
// 这能直观体现"波及层级"，但不还原严格父子边（同层符号的父节点未追溯）。
// 若需精确路径树，需增强 API 返回 parent 信息。
func displayImpactTree(w io.Writer, resp *client.TransitiveResponse, direction string, duration time.Duration) {
	fmt.Fprint(w, renderImpactTree(resp, direction, duration))
}

// renderImpactTree 生成树形输出的字符串（纯函数，便于测试）。
func renderImpactTree(resp *client.TransitiveResponse, direction string, duration time.Duration) string {
	header := "callees"
	if direction == "callers" {
		header = "callers"
	}

	if resp == nil || len(resp.Symbols) == 0 {
		return fmt.Sprintf("No transitive %s found.\n", header)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Transitive %s (max depth=%d, %d symbols, %s)\n\n",
		header, resp.Depth, resp.Total, duration.Truncate(time.Millisecond))

	// 按 depth 分组
	byDepth := make(map[int][]client.ReachableSymbol)
	maxDepth := 0
	for _, s := range resp.Symbols {
		byDepth[s.Depth] = append(byDepth[s.Depth], s)
		if s.Depth > maxDepth {
			maxDepth = s.Depth
		}
	}

	for d := 1; d <= maxDepth; d++ {
		symbols := byDepth[d]
		if len(symbols) == 0 {
			continue
		}
		// 同层按符号名排序，输出稳定
		sort.Slice(symbols, func(i, j int) bool {
			return symbols[i].Name < symbols[j].Name
		})

		noun := "call"
		if direction == "callers" {
			noun = "caller"
		}
		fmt.Fprintf(&b, "%s (depth=%d, %d):\n", noun, d, len(symbols))
		for _, s := range symbols {
			fmt.Fprintf(&b, "  ├── %s [%s]  %s\n", s.Name, s.Kind, s.FilePath)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}
