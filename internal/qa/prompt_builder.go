// Package qa 把检索上下文组装为可直接喂给 LLM 的 Markdown prompt，
// 并提供结构化 JSON 视图。本层不碰数据库、不碰 HTTP。
package qa

import (
	"fmt"
	"strings"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// PromptBuildOptions 控制 prompt 拼接。
type PromptBuildOptions struct {
	MaxTokens     int  // 软上限，默认 8000（按 4 字符 ≈ 1 token 估算）
	IncludeSource bool // 是否内联源码片段
}

// DefaultPromptBuildOptions 返回默认配置。
func DefaultPromptBuildOptions() PromptBuildOptions {
	return PromptBuildOptions{MaxTokens: 8000}
}

// BuildPrompt 把 ContextBlock[] 拼成 Markdown prompt。
// sources 是 chunkID → 源码文本映射（IncludeSource 时传入，可为 nil）。
// 返回 prompt 文本和是否被截断。
// 截断策略：超限时优先保留高 similarity 的 block；先砍低分 block 的图谱邻居，再砍整个低分 block。
// 注意：调用方需保证 blocks 已按 Similarity 降序排列（retrieval 层返回即降序）。
func BuildPrompt(query string, repoIDs []string, blocks []retrieval.ContextBlock, sources map[string]string, opts PromptBuildOptions) (string, bool) {
	if opts.MaxTokens == 0 {
		opts = DefaultPromptBuildOptions()
	}

	var sb strings.Builder
	sb.WriteString("# Code Context\n\n")
	sb.WriteString("## Question\n")
	sb.WriteString(query + "\n\n")

	if len(repoIDs) > 0 {
		sb.WriteString("## Repositories\n")
		sb.WriteString(strings.Join(repoIDs, ", ") + "\n\n")
	}

	sb.WriteString("## Relevant Symbols\n\n")

	charBudget := opts.MaxTokens * 4
	truncated := false
	for i, b := range blocks {
		section := formatBlockSection(i+1, b, sources, opts.IncludeSource)
		// 预算检查：若加上这段会超，先尝试去掉图谱邻居再拼
		if len(section) > charBudget {
			stripped := b
			stripped.Callers = nil
			stripped.Callees = nil
			section = formatBlockSection(i+1, stripped, sources, opts.IncludeSource)
			truncated = true
		}
		if len(section) > charBudget {
			// 整个 block 放不下，跳过低分的（blocks 已按 similarity 降序）
			truncated = true
			continue
		}
		sb.WriteString(section)
		charBudget -= len(section)
	}

	return sb.String(), truncated
}

// formatBlockSection 渲染单个 block 的 Markdown 段落。
func formatBlockSection(idx int, b retrieval.ContextBlock, sources map[string]string, includeSource bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %d. %s (similarity: %.2f)\n", idx, b.Symbol.Name, b.Similarity)
	if b.Symbol.FilePath != "" {
		fmt.Fprintf(&sb, "- **File**: `%s`\n", b.Symbol.FilePath)
	}
	if b.Symbol.Signature != "" {
		fmt.Fprintf(&sb, "- **Signature**: `%s`\n", b.Symbol.Signature)
	}
	if b.Symbol.Docstring != "" {
		fmt.Fprintf(&sb, "- **Docstring**: %s\n", b.Symbol.Docstring)
	}
	if len(b.Callers) > 0 {
		sb.WriteString("- **Called by**:\n")
		for _, c := range b.Callers {
			fmt.Fprintf(&sb, "  - `%s` (%s)\n", c.Name, c.FilePath)
		}
	}
	if len(b.Callees) > 0 {
		sb.WriteString("- **Calls**:\n")
		for _, c := range b.Callees {
			fmt.Fprintf(&sb, "  - `%s` (%s)\n", c.Name, c.FilePath)
		}
	}
	if includeSource && sources != nil {
		if src, ok := sources[b.ChunkID]; ok && src != "" {
			fmt.Fprintf(&sb, "\n```\n%s\n```\n", src)
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
