// Package qa 把检索上下文组装为可直接喂给 LLM 的 Markdown prompt，
// 并提供结构化 JSON 视图。本层不碰数据库、不碰 HTTP。
package qa

import (
	"context"
	"fmt"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// AskRequest 是 QA 端点的请求。
type AskRequest struct {
	Query         string
	RepoIDs       []string
	Language      string
	Kind          []string
	Mode          string
	Limit         int
	IncludeSource bool
	ExpandCallers bool
	ExpandCallees bool
}

// AskResponse 是 QA 端点的响应。
type AskResponse struct {
	Query     string             `json:"query"`
	Blocks    []ContextBlockJSON `json:"blocks"`
	Prompt    string             `json:"prompt"`
	Truncated bool               `json:"truncated"`
	ChunkIDs  []string           `json:"chunk_ids"`
}

// ContextBlockJSON 是结构化 JSON 视图。
type ContextBlockJSON struct {
	Symbol     SymbolJSON   `json:"symbol"`
	Similarity float64      `json:"similarity"`
	MatchMode  string       `json:"match_mode"`
	Callers    []SymbolJSON `json:"callers"`
	Callees    []SymbolJSON `json:"callees"`
	ChunkID    string       `json:"chunk_id"`
	Source     string       `json:"source,omitempty"`
}

// SymbolJSON 是符号的 JSON 视图。
type SymbolJSON struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Signature string `json:"signature,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Language  string `json:"language,omitempty"`
	Docstring string `json:"docstring,omitempty"`
}

// SourceFetcher 按 chunk_id 批量取源码（IncludeSource 时用）。
type SourceFetcher interface {
	GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// Service 是 QA 编排接口。
type Service interface {
	Ask(ctx context.Context, req AskRequest) (*AskResponse, error)
}

type service struct {
	retriever     retrieval.Retriever
	sourceFetcher SourceFetcher
	promptOpts    PromptBuildOptions
}

// NewService 创建 QA service。
func NewService(r retrieval.Retriever, sf SourceFetcher, opts PromptBuildOptions) Service {
	if opts.MaxTokens == 0 {
		opts = DefaultPromptBuildOptions()
	}
	return &service{retriever: r, sourceFetcher: sf, promptOpts: opts}
}

// Ask 执行 QA 编排。
func (s *service) Ask(ctx context.Context, req AskRequest) (*AskResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	blocks, err := s.retriever.Query(ctx, retrieval.RetrievalRequest{
		Query:         req.Query,
		RepoIDs:       req.RepoIDs,
		Language:      req.Language,
		Kind:          req.Kind,
		Mode:          req.Mode,
		Limit:         req.Limit,
		ExpandHops:    1,
		ExpandCallers: req.ExpandCallers,
		ExpandCallees: req.ExpandCallees,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 按需取源码
	sources := map[string]string{}
	if req.IncludeSource && s.sourceFetcher != nil {
		chunkIDs := collectChunkIDs(blocks)
		if len(chunkIDs) > 0 {
			if fetched, err := s.sourceFetcher.GetByVectorIDs(ctx, chunkIDs); err == nil {
				sources = fetched
			}
		}
	}

	// 拼 prompt（IncludeSource 由 opts 标记，源码通过 sources 参数传入）
	promptOpts := s.promptOpts
	promptOpts.IncludeSource = req.IncludeSource
	prompt, truncated := BuildPrompt(req.Query, req.RepoIDs, blocks, sources, promptOpts)

	// 组装 JSON 响应
	resp := &AskResponse{
		Query:     req.Query,
		Blocks:    toBlockJSONs(blocks, sources),
		Prompt:    prompt,
		Truncated: truncated,
		ChunkIDs:  collectChunkIDs(blocks),
	}
	return resp, nil
}

func collectChunkIDs(blocks []retrieval.ContextBlock) []string {
	seen := map[string]bool{}
	var ids []string
	for _, b := range blocks {
		if b.ChunkID != "" && !seen[b.ChunkID] {
			seen[b.ChunkID] = true
			ids = append(ids, b.ChunkID)
		}
	}
	return ids
}

func toBlockJSONs(blocks []retrieval.ContextBlock, sources map[string]string) []ContextBlockJSON {
	result := make([]ContextBlockJSON, 0, len(blocks))
	for _, b := range blocks {
		result = append(result, ContextBlockJSON{
			Symbol:     toSymbolJSON(b.Symbol),
			Similarity: b.Similarity,
			MatchMode:  b.MatchMode,
			Callers:    toSymbolJSONs(b.Callers),
			Callees:    toSymbolJSONs(b.Callees),
			ChunkID:    b.ChunkID,
			Source:     sources[b.ChunkID],
		})
	}
	return result
}

func toSymbolJSON(s retrieval.ContextSymbol) SymbolJSON {
	return SymbolJSON{
		SymbolID: s.SymbolID, Name: s.Name, Kind: s.Kind,
		Signature: s.Signature, FilePath: s.FilePath,
		Language: s.Language, Docstring: s.Docstring,
	}
}

func toSymbolJSONs(ss []retrieval.ContextSymbol) []SymbolJSON {
	r := make([]SymbolJSON, 0, len(ss))
	for _, s := range ss {
		r = append(r, toSymbolJSON(s))
	}
	return r
}
