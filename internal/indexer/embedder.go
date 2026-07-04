package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// Embedder interface defines methods for generating and storing embeddings
type Embedder interface {
	// GenerateEmbedding creates a vector for text content
	GenerateEmbedding(ctx context.Context, content string) ([]float32, error)

	// EmbedSymbols generates embeddings for symbols with docstrings
	EmbedSymbols(ctx context.Context, symbols []schema.Symbol) (*EmbedResult, error)

	// BatchEmbed processes multiple texts in a single API call
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingInput 是单个待嵌入单元，由 Chunker 产出。
// 一个 symbol 可被切分为多个 chunk（多粒度 embedding），每个 chunk
// 携带归属的 entity_id（默认为 symbol_id）与 chunk_index。
type EmbeddingInput struct {
	EntityID   string
	Content    string
	ChunkIndex int
}

// Chunker 把符号切分为待嵌入的文本单元。
//
// 当前唯一实现 SymbolChunker 维持原有"逐 symbol 单向量"行为；
// 下一站检索质量优化可新增 MultiGranularityChunker（按文件/按代码块），
// 通过 SetChunker 注入，无需改动 EmbedSymbols 主流程。
type Chunker interface {
	// Chunk 将符号转换为待嵌入输入列表。
	// 返回空 content 的项应被跳过（与原 buildSymbolContent 行为一致）。
	Chunk(symbols []schema.Symbol) []EmbeddingInput
}

// SymbolChunker 是默认实现：每个有内容的 symbol 产出一条 EmbeddingInput，
// 内容由 signature/docstring/semantic_summary 拼接（与原 buildSymbolContent 一致）。
type SymbolChunker struct{}

// Chunk 实现 Chunker 接口。
func (SymbolChunker) Chunk(symbols []schema.Symbol) []EmbeddingInput {
	out := make([]EmbeddingInput, 0, len(symbols))
	for _, s := range symbols {
		content := buildSymbolContent(s)
		if content == "" {
			continue
		}
		out = append(out, EmbeddingInput{
			EntityID:   s.SymbolID,
			Content:    content,
			ChunkIndex: 0,
		})
	}
	return out
}

// CodeBlockChunker 把同文件中行号相邻的符号合并为代码块级 chunk。
//
// 动机：单个符号的语义常不完整（一个业务逻辑分散在相邻的多个函数/类里）。
// 把相邻符号拼接为一个 chunk，让向量携带"代码块"的整体语义，提升召回覆盖。
//
// 分块策略：按 file_id 分组 → 组内按 start_line 排序 → 相邻符号 start_line
// 差 ≤ GapThreshold 行则合并为一块。每块拼接其所有符号的 signature/docstring/
// summary。entity_id 取块内"主要符号"（function > class > interface > 其他），
// 保证检索时 JOIN symbols 能落到该块最具代表性的符号上。
type CodeBlockChunker struct {
	// GapThreshold 是合并相邻符号的最大行距，超过则开新块。默认 30。
	GapThreshold int
}

// NewCodeBlockChunker 创建代码块级 chunker。
func NewCodeBlockChunker(gapThreshold int) *CodeBlockChunker {
	if gapThreshold <= 0 {
		gapThreshold = 30
	}
	return &CodeBlockChunker{GapThreshold: gapThreshold}
}

// Chunk 实现 Chunker 接口。
func (c CodeBlockChunker) Chunk(symbols []schema.Symbol) []EmbeddingInput {
	if len(symbols) == 0 {
		return nil
	}

	// 1. 按 file_id 分组
	files := make(map[string][]schema.Symbol)
	for _, s := range symbols {
		files[s.FileID] = append(files[s.FileID], s)
	}

	outputs := make([]EmbeddingInput, 0, len(symbols))
	// 2. 每个文件内排序 + 分块
	for fileID, syms := range files {
		sort.Slice(syms, func(i, j int) bool {
			return syms[i].Span.StartLine < syms[j].Span.StartLine
		})

		var currentBlock []schema.Symbol
		var lastEndLine int
		flush := func() {
			if len(currentBlock) == 0 {
				return
			}
			outputs = append(outputs, c.blockToInput(currentBlock))
			currentBlock = nil
		}

		for _, s := range syms {
			content := buildSymbolContent(s)
			if content == "" {
				continue // 跳过无内容符号
			}
			if len(currentBlock) > 0 && s.Span.StartLine-lastEndLine > c.GapThreshold {
				flush() // 间距超阈值，开新块
			}
			currentBlock = append(currentBlock, s)
			lastEndLine = s.Span.EndLine
		}
		flush()
		_ = fileID // fileID 用于分组，块内通过 entity symbol 关联
	}
	return outputs
}

// blockToInput 把一个符号块转为单条 EmbeddingInput。
// entity_id 取主要符号；content 拼接块内所有符号内容。
func (c CodeBlockChunker) blockToInput(block []schema.Symbol) EmbeddingInput {
	var parts []string
	primary := block[0]
	primaryRank := symbolRank(block[0].Kind)
	for _, s := range block {
		parts = append(parts, buildSymbolContent(s))
		// 选主要符号：function/class/interface 优先
		if r := symbolRank(s.Kind); r < primaryRank {
			primary = s
			primaryRank = r
		}
	}
	return EmbeddingInput{
		EntityID:   primary.SymbolID,
		Content:    strings.Join(parts, "\n---\n"),
		ChunkIndex: 0,
	}
}

// symbolRank 返回符号作为块代表的优先级（越小越优先）。
func symbolRank(kind schema.SymbolKind) int {
	switch kind {
	case schema.SymbolFunction:
		return 0
	case schema.SymbolClass:
		return 1
	case schema.SymbolInterface:
		return 2
	case schema.SymbolVariable:
		return 3
	default:
		return 4
	}
}

// buildSymbolContent constructs content for embedding from symbol.
// 从 OpenAIEmbedder 方法提取为包级函数，供 SymbolChunker 复用。
func buildSymbolContent(symbol schema.Symbol) string {
	var parts []string
	if symbol.Signature != "" {
		parts = append(parts, symbol.Signature)
	}
	if symbol.Docstring != "" {
		parts = append(parts, symbol.Docstring)
	}
	if symbol.SemanticSummary != "" {
		parts = append(parts, symbol.SemanticSummary)
	}
	return strings.Join(parts, "\n")
}

// EmbedderConfig contains configuration options for the embedder
type EmbedderConfig struct {
	// Backend type: "openai" or "local"
	Backend string `json:"backend"`

	// API endpoint URL (for OpenAI-compatible APIs)
	APIEndpoint string `json:"api_endpoint"`

	// API key for authentication (optional for local)
	APIKey string `json:"api_key"`

	// Model name to use for embeddings
	Model string `json:"model"`

	// Expected embedding dimensions
	Dimensions int `json:"dimensions"`

	// Batch size for API calls
	BatchSize int `json:"batch_size"`

	// Rate limiting: max requests per second
	MaxRequestsPerSecond int `json:"max_requests_per_second"`

	// Retry configuration
	MaxRetries     int           `json:"max_retries"`
	BaseRetryDelay time.Duration `json:"base_retry_delay"`
	MaxRetryDelay  time.Duration `json:"max_retry_delay"`

	// HTTP client timeout
	Timeout time.Duration `json:"timeout"`
}

// DefaultEmbedderConfig returns default configuration
// Note: Adjust Dimensions based on your model:
// - text-embedding-qwen3-embedding-0.6b: 1024 dimensions
// - nomic-embed-text: 768 dimensions
// - text-embedding-3-small (OpenAI): 1536 dimensions
func DefaultEmbedderConfig() *EmbedderConfig {
	return &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           1024, // text-embedding-qwen3-embedding-0.6b uses 1024 dimensions
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
		MaxRetries:           3,
		BaseRetryDelay:       100 * time.Millisecond,
		MaxRetryDelay:        5 * time.Second,
		Timeout:              30 * time.Second,
	}
}

// OpenAIEmbedder implements the Embedder interface using OpenAI-compatible API
type OpenAIEmbedder struct {
	config      *EmbedderConfig
	httpClient  *http.Client
	vectorRepo  *models.VectorRepository
	rateLimiter *rateLimiter
	chunker     Chunker
	mu          sync.Mutex
}

// NewOpenAIEmbedder creates a new OpenAI-compatible embedder
func NewOpenAIEmbedder(config *EmbedderConfig, vectorRepo *models.VectorRepository) *OpenAIEmbedder {
	if config == nil {
		config = DefaultEmbedderConfig()
	}

	return &OpenAIEmbedder{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		vectorRepo:  vectorRepo,
		rateLimiter: newRateLimiter(config.MaxRequestsPerSecond),
		chunker:     SymbolChunker{},
	}
}

// SetChunker 替换符号切分策略，仅供检索质量优化时注入多粒度实现。
// 传入 nil 等价于恢复默认 SymbolChunker。
func (e *OpenAIEmbedder) SetChunker(c Chunker) {
	if c == nil {
		e.chunker = SymbolChunker{}
	} else {
		e.chunker = c
	}
}

// EmbedResult contains the results of an embedding operation
type EmbedResult struct {
	VectorsCreated int           `json:"vectors_created"`
	Duration       time.Duration `json:"duration"`
	Errors         []EmbedError  `json:"errors,omitempty"`
}

// EmbedError represents an error that occurred during embedding
type EmbedError struct {
	EntityID string `json:"entity_id"`
	Message  string `json:"message"`
}

// GenerateEmbedding creates a vector for text content
func (e *OpenAIEmbedder) GenerateEmbedding(ctx context.Context, content string) ([]float32, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Wait for rate limiter
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(e.config.BaseRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > e.config.MaxRetryDelay {
				delay = e.config.MaxRetryDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embedding, err := e.callEmbeddingAPI(ctx, []string{content})
		if err == nil && len(embedding) > 0 {
			return embedding[0], nil
		}
		lastErr = err

		// Don't retry on certain errors
		if !e.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed to generate embedding after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// BatchEmbed processes multiple texts in a single API call
func (e *OpenAIEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Wait for rate limiter
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(e.config.BaseRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > e.config.MaxRetryDelay {
				delay = e.config.MaxRetryDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embeddings, err := e.callEmbeddingAPI(ctx, texts)
		if err == nil {
			return embeddings, nil
		}
		lastErr = err

		// Don't retry on certain errors
		if !e.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed to batch embed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// EmbedSymbols generates embeddings for symbols with docstrings
func (e *OpenAIEmbedder) EmbedSymbols(ctx context.Context, symbols []schema.Symbol) (*EmbedResult, error) {
	startTime := time.Now()
	result := &EmbedResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	// 通过 Chunker 把符号切分为待嵌入单元。
	// 默认 SymbolChunker 维持原"逐 symbol 单向量"行为；
	// 下一站可通过 SetChunker 注入多粒度实现。
	inputs := e.chunker.Chunk(symbols)
	if len(inputs) == 0 {
		return result, nil
	}

	// Process in batches
	for i := 0; i < len(inputs); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(inputs) {
			end = len(inputs)
		}

		batch := inputs[i:end]
		batchContents := make([]string, len(batch))
		for k, in := range batch {
			batchContents[k] = in.Content
		}

		// Generate embeddings for batch
		embeddings, err := e.BatchEmbed(ctx, batchContents)
		if err != nil {
			// Log error but continue with other batches
			for _, in := range batch {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: in.EntityID,
					Message:  fmt.Sprintf("failed to generate embedding: %v", err),
				})
			}
			continue
		}

		// Validate dimensions
		for j, embedding := range embeddings {
			if len(embedding) != e.config.Dimensions {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: batch[j].EntityID,
					Message:  fmt.Sprintf("invalid embedding dimensions: expected %d, got %d", e.config.Dimensions, len(embedding)),
				})
				continue
			}

			// Store embedding
			vector := &models.Vector{
				VectorID:   uuid.New().String(),
				EntityID:   batch[j].EntityID,
				EntityType: "symbol",
				Embedding:  embedding,
				Content:    batch[j].Content,
				Model:      e.config.Model,
				ChunkIndex: batch[j].ChunkIndex,
			}

			err := e.vectorRepo.Create(ctx, vector)
			if err != nil {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: batch[j].EntityID,
					Message:  fmt.Sprintf("failed to store embedding: %v", err),
				})
			} else {
				result.VectorsCreated++
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// callEmbeddingAPI makes the actual API call to generate embeddings
func (e *OpenAIEmbedder) callEmbeddingAPI(ctx context.Context, texts []string) ([][]float32, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"input": texts,
		"model": e.config.Model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", e.config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.config.APIKey))
	}

	// Make request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(apiResp.Data))
	for i, data := range apiResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// isRetryableError determines if an error is retryable
func (e *OpenAIEmbedder) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are retryable
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") {
		return true
	}

	// Rate limit errors are retryable
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Server errors are retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	return false
}

// OpenAIEmbeddingResponse represents the API response structure
type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	rate       int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(requestsPerSecond int) *rateLimiter {
	return &rateLimiter{
		rate:       requestsPerSecond,
		interval:   time.Second,
		tokens:     requestsPerSecond,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (rl *rateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill)
		if elapsed >= rl.interval {
			rl.tokens = rl.rate
			rl.lastRefill = now
		}

		// Check if token available
		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time
		waitTime := rl.interval - elapsed
		rl.mu.Unlock()

		// Wait or check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}
