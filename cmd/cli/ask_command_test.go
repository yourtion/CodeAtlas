package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// fixedQAResponse is the canned response returned by the mock API server.
var fixedQAResponse = client.QAResponse{
	Query: "how does auth work",
	Blocks: []client.QABlock{
		{
			Symbol: client.QASymbol{
				SymbolID:  "s1",
				Name:      "Login",
				Kind:      "function",
				Signature: "func Login(u, p string)",
				FilePath:  "auth/login.go",
			},
			Similarity: 0.91,
			MatchMode:  "hybrid",
			Callers: []client.QASymbol{
				{SymbolID: "c1", Name: "handleLogin", Kind: "function", FilePath: "server.go"},
			},
			ChunkID: "chk1",
		},
	},
	Prompt:    "# Code Context\n\n## Login\nfunc Login(u, p string)\n",
	Truncated: false,
	ChunkIDs:  []string{"chk1"},
}

// startMockQAServer starts an httptest server that returns fixedQAResponse.
// It records the last received request body into lastReq for assertion.
func startMockQAServer(t *testing.T, lastReq *map[string]interface{}) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/qa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if lastReq != nil {
			*lastReq = nil
			_ = json.Unmarshal(body, lastReq)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fixedQAResponse)
	})
	return httptest.NewServer(mux)
}

// runAskCommand builds a standalone cli.App with the ask command and runs it
// against the given args. Stdout is captured and returned.
func runAskCommand(t *testing.T, args []string) (stdout string, exitErr error) {
	t.Helper()
	app := &cli.App{
		Name:     "codeatlas-test",
		Commands: []*cli.Command{createAskCommand()},
	}

	// Capture stdout.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	defer func() {
		os.Stdout = old
	}()

	exitErr = app.Run(args)

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String(), exitErr
}

// TestExecuteAsk_DefaultPromptToStdout verifies the default output is the
// assembled prompt text printed to stdout.
func TestExecuteAsk_DefaultPromptToStdout(t *testing.T) {
	srv := startMockQAServer(t, nil)
	defer srv.Close()

	out, err := runAskCommand(t, []string{
		"codeatlas", "ask",
		"--question", "how does auth work",
		"--api-url", srv.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, fixedQAResponse.Prompt) {
		t.Errorf("stdout should contain the prompt, got: %q", out)
	}
	// Should NOT be JSON (no top-level query/prompt keys as a JSON object).
	if strings.Contains(out, `"prompt":`) {
		t.Errorf("default output should be plain prompt, not JSON, got: %q", out)
	}
}

// TestExecuteAsk_OutputFile verifies --output writes the prompt to a file.
func TestExecuteAsk_OutputFile(t *testing.T) {
	srv := startMockQAServer(t, nil)
	defer srv.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "prompt.md")

	out, err := runAskCommand(t, []string{
		"codeatlas", "ask",
		"--question", "how does auth work",
		"--api-url", srv.URL,
		"--output", outPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nothing meaningful printed to stdout when writing to file.
	if strings.Contains(out, fixedQAResponse.Prompt) {
		t.Errorf("stdout should be empty when --output set, got: %q", out)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if string(got) != fixedQAResponse.Prompt {
		t.Errorf("file content = %q, want %q", string(got), fixedQAResponse.Prompt)
	}
}

// TestExecuteAsk_JSONOutput verifies --json prints the full JSON response.
func TestExecuteAsk_JSONOutput(t *testing.T) {
	srv := startMockQAServer(t, nil)
	defer srv.Close()

	out, err := runAskCommand(t, []string{
		"codeatlas", "ask",
		"--question", "how does auth work",
		"--api-url", srv.URL,
		"--json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed client.QAResponse
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("stdout should be valid JSON QAResponse: %v; got: %s", err, out)
	}
	if parsed.Query != fixedQAResponse.Query {
		t.Errorf("parsed query = %q, want %q", parsed.Query, fixedQAResponse.Query)
	}
	if parsed.Prompt != fixedQAResponse.Prompt {
		t.Errorf("parsed prompt = %q, want %q", parsed.Prompt, fixedQAResponse.Prompt)
	}
	if len(parsed.Blocks) != 1 || parsed.Blocks[0].Symbol.Name != "Login" {
		t.Errorf("parsed blocks mismatch: %+v", parsed.Blocks)
	}
}

// TestExecuteAsk_RepoRepeatable verifies --repo is repeatable and the
// collected repo IDs are forwarded in the request body.
func TestExecuteAsk_RepoRepeatable(t *testing.T) {
	var lastReq map[string]interface{}
	srv := startMockQAServer(t, &lastReq)
	defer srv.Close()

	// Mix long and short forms across separate invocations to prove both
	// alias and full name append to the same slice.
	_, err := runAskCommand(t, []string{
		"codeatlas", "ask",
		"--question", "how does auth work",
		"--api-url", srv.URL,
		"--repo", "repo-a",
		"--repo", "repo-b",
		"--repo", "repo-c",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repoIDs, ok := lastReq["repo_ids"].([]interface{})
	if !ok {
		t.Fatalf("request body should contain repo_ids array; got: %#v", lastReq)
	}
	if len(repoIDs) != 3 {
		t.Fatalf("expected 3 repo_ids, got %d: %v", len(repoIDs), repoIDs)
	}
	got := []string{}
	for _, v := range repoIDs {
		got = append(got, v.(string))
	}
	want := []string{"repo-a", "repo-b", "repo-c"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("repo_ids[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestExecuteAsk_RepoAlias verifies the -r short alias also collects values.
func TestExecuteAsk_RepoAlias(t *testing.T) {
	var lastReq map[string]interface{}
	srv := startMockQAServer(t, &lastReq)
	defer srv.Close()

	_, err := runAskCommand(t, []string{
		"codeatlas", "ask",
		"--question", "how does auth work",
		"--api-url", srv.URL,
		"-r", "repo-x",
		"-r", "repo-y",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repoIDs, ok := lastReq["repo_ids"].([]interface{})
	if !ok || len(repoIDs) != 2 {
		t.Fatalf("expected 2 repo_ids via -r alias; got: %#v", lastReq["repo_ids"])
	}
	if repoIDs[0] != "repo-x" || repoIDs[1] != "repo-y" {
		t.Errorf("repo_ids = %v, want [repo-x repo-y]", repoIDs)
	}
}
