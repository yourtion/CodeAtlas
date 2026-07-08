package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/internal/quality/fixtures"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// errGateFailed 表示评估门禁未通过（有指标低于阈值）。
// runEval 返回此 error，由 main 据此 exit 1，避免 outputReport 里直接 os.Exit 绕过 defer。
var errGateFailed = errors.New("quality gate failed")

// createEvalCommand 创建质量评估命令。
//
// 评估依赖图与检索质量指标，支持两种模式：
//   - --repo <id>：评估真实仓库（结构断言，建基线）
//   - --fixtures：评估 fixture 真值集（recall/precision/MRR，门禁用）
func createEvalCommand() *cli.Command {
	return &cli.Command{
		Name:  "eval",
		Usage: "Evaluate code knowledge graph and retrieval quality",
		Description: `Evaluate dependency graph and retrieval quality metrics.

EXAMPLES:
   # Evaluate a real repository (structural metrics, baseline)
   codeatlas eval --repo <repo_id> --db "host=localhost port=5432 user=codeatlas dbname=codeatlas sslmode=disable"

   # Evaluate fixture ground truth (recall/precision/MRR, gating)
   codeatlas eval --fixtures --db "..." --format json

   # Only run one category
   codeatlas eval --repo <repo_id> --only graph
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo",
				Usage: "Repository ID to evaluate (repo mode)",
			},
			&cli.BoolFlag{
				Name:  "fixtures",
				Usage: "Evaluate fixture ground truth (fixture mode)",
			},
			&cli.StringFlag{
				Name:    "db",
				Usage:   "PostgreSQL connection string",
				EnvVars: []string{"DATABASE_URL", "DB_DSN"},
			},
			&cli.StringFlag{
				Name:  "only",
				Usage: "Run only one category: graph | retrieval (empty = all)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: text | json",
				Value: "text",
			},
		},
		Action: runEval,
	}
}

func runEval(c *cli.Context) error {
	repoID := c.String("repo")
	fixturesMode := c.Bool("fixtures")

	if repoID != "" && fixturesMode {
		return fmt.Errorf("--repo and --fixtures are mutually exclusive")
	}
	if repoID == "" && !fixturesMode {
		return fmt.Errorf("must specify either --repo or --fixtures")
	}

	dsn := c.String("db")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOrEval("DB_HOST", "localhost"), envOrEval("DB_PORT", "5432"),
			envOrEval("DB_USER", "codeatlas"), envOrEval("DB_PASSWORD", "codeatlas"),
			envOrEval("DB_NAME", "codeatlas"))
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	modelsDB := &models.DB{DB: db}
	edgeRepo := models.NewEdgeRepository(modelsDB)
	symbolRepo := models.NewSymbolRepository(modelsDB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	only := c.String("only")
	format := c.String("format")
	ctx := context.Background()

	if fixturesMode {
		// fixture 模式：评估已索引的 fixture 仓库。
		// 需要 --repo 指定已索引 fixture 的 repoID（CLI 不自行索引，
		// 用 codeatlas index 索引 fixture 后再用 --repo + --fixtures 评估）。
		if repoID == "" {
			return fmt.Errorf("--fixtures 模式需要 --repo 指定已索引的 fixture repoID。\n" +
				"先索引 fixture：codeatlas index --path <fixture_dir> --server <url>\n" +
				"再评估：codeatlas eval --fixtures --repo <repoID> --db ...")
		}
		// 合并依赖图真值
		truth := &quality.GraphGroundTruth{FixtureFile: "merged"}
		for _, gt := range fixtures.CallAnalysisGroundTruth {
			truth.Edges = append(truth.Edges, gt.Edges...)
			truth.Chains = append(truth.Chains, gt.Chains...)
		}
		graphEval := quality.NewGraphEvaluator(fetcher, truth)

		cfg := quality.EvaluateConfig{
			Mode:         quality.EvalModeFixture,
			FixtureSet:   "call_analysis",
			RepoID:       repoID,
			RunRetrieval: false, // CLI 暂不支持 retrieval（需 embedder）
		}
		if only == "retrieval" {
			fmt.Fprintln(os.Stderr, "注意：retrieval 评估需 embedder 配置，请用 make test-integration 跑。")
		}

		report, err := quality.Evaluate(ctx, cfg, graphEval, nil)
		if err != nil {
			return err
		}
		return outputReport(report, format)
	}

	// repo 模式
	graphEval := quality.NewGraphEvaluator(fetcher, nil)
	cfg := quality.EvaluateConfig{
		Mode:   quality.EvalModeRepo,
		RepoID: repoID,
	}
	report, err := quality.Evaluate(ctx, cfg, graphEval, nil)
	if err != nil {
		return err
	}
	return outputReport(report, format)
}

func outputReport(report *quality.Report, format string) error {
	if format == "json" {
		data, err := report.JSONMarshal()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// text 格式
	fmt.Println("CodeAtlas Quality Report")
	fmt.Println("========================")
	fmt.Printf("Mode: %s", report.Mode)
	if report.RepoID != "" {
		fmt.Printf("  RepoID: %s", report.RepoID)
	}
	if report.FixtureSet != "" {
		fmt.Printf("  FixtureSet: %s", report.FixtureSet)
	}
	fmt.Println()

	for _, cat := range []quality.MetricCategory{quality.CategoryGraph, quality.CategoryRetrieval} {
		hasCat := false
		for _, m := range report.Metrics {
			if m.Category == cat {
				hasCat = true
				break
			}
		}
		if !hasCat {
			continue
		}
		fmt.Printf("\n== %s Metrics ==\n", titleCaseEval(string(cat)))
		for _, m := range report.Metrics {
			if m.Category != cat {
				continue
			}
			mark := "✓"
			if !m.Passed {
				mark = "✗"
			}
			if m.Threshold == 0 {
				fmt.Printf("  %-35s %.4f  (仅观察)  %s\n", m.Name, m.Value, mark)
			} else {
				op := "≥"
				if !m.HigherIsBetter {
					op = "≤"
				}
				fmt.Printf("  %-35s %.4f  (%s%.2f)  %s\n", m.Name, m.Value, op, m.Threshold, mark)
			}
		}
	}

	fmt.Printf("\nSummary: %d passed, %d failed, %d observed\n",
		report.Summary.Passed, report.Summary.Failed, report.Summary.NoThreshold)

	if report.Summary.Failed > 0 {
		return errGateFailed
	}
	return nil
}

func envOrEval(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func titleCaseEval(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
