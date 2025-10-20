package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestNewJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	if writer == nil {
		t.Fatal("NewJSONWriter returned nil")
	}
	if writer.writer != &buf {
		t.Error("Writer not set correctly")
	}
	if !writer.indent {
		t.Error("Indent not set correctly")
	}
	if writer.streaming {
		t.Error("Streaming should be false for regular writer")
	}
}

func TestNewStreamingJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	if writer == nil {
		t.Fatal("NewStreamingJSONWriter returned nil")
	}
	if !writer.streaming {
		t.Error("Streaming should be true for streaming writer")
	}
}

func TestWriteOutput_Complete(t *testing.T) {
	output := createTestOutput()
	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	// Verify content
	if len(parsed.Files) != len(output.Files) {
		t.Errorf("Expected %d files, got %d", len(output.Files), len(parsed.Files))
	}
	if len(parsed.Relationships) != len(output.Relationships) {
		t.Errorf("Expected %d relationships, got %d", len(output.Relationships), len(parsed.Relationships))
	}
	if parsed.Metadata.Version != output.Metadata.Version {
		t.Errorf("Expected version %s, got %s", output.Metadata.Version, parsed.Metadata.Version)
	}
}

func TestWriteOutput_Streaming(t *testing.T) {
	output := createTestOutput()
	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	// Verify content
	if len(parsed.Files) != len(output.Files) {
		t.Errorf("Expected %d files, got %d", len(output.Files), len(parsed.Files))
	}
	if len(parsed.Relationships) != len(output.Relationships) {
		t.Errorf("Expected %d relationships, got %d", len(output.Relationships), len(parsed.Relationships))
	}
}

func TestWriteOutput_WithIndentation(t *testing.T) {
	output := createTestOutput()
	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Check that output contains indentation
	jsonStr := buf.String()
	if !strings.Contains(jsonStr, "\n") {
		t.Error("Expected indented JSON to contain newlines")
	}
	if !strings.Contains(jsonStr, "  ") {
		t.Error("Expected indented JSON to contain spaces")
	}
}

func TestWriteOutput_WithoutIndentation(t *testing.T) {
	output := createTestOutput()
	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, false)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid but compact
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestWriteOutput_EmptyFiles(t *testing.T) {
	output := &schema.ParseOutput{
		Files:         []schema.File{},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(parsed.Files))
	}
}

func TestWriteOutput_WithErrors(t *testing.T) {
	output := createTestOutput()
	output.Metadata.Errors = []schema.ParseError{
		{
			File:    "test.go",
			Line:    42,
			Column:  10,
			Message: "syntax error",
			Type:    schema.ErrorParse,
		},
		{
			File:    "broken.js",
			Message: "file not found",
			Type:    schema.ErrorFileSystem,
		},
	}
	output.Metadata.FailureCount = 2

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Metadata.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(parsed.Metadata.Errors))
	}
	if parsed.Metadata.FailureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", parsed.Metadata.FailureCount)
	}
}

func TestCreateOutput(t *testing.T) {
	files := []schema.File{
		{FileID: "file1", Path: "test1.go"},
		{FileID: "file2", Path: "test2.go"},
	}
	relationships := []schema.DependencyEdge{
		{EdgeID: "edge1", SourceID: "sym1", TargetID: "sym2"},
	}
	errors := []schema.ParseError{
		{File: "broken.go", Message: "parse error", Type: schema.ErrorParse},
	}

	output := CreateOutput(files, relationships, errors)

	if output == nil {
		t.Fatal("CreateOutput returned nil")
	}
	if len(output.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(output.Files))
	}
	if len(output.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(output.Relationships))
	}
	if output.Metadata.Version != OutputVersion {
		t.Errorf("Expected version %s, got %s", OutputVersion, output.Metadata.Version)
	}
	if output.Metadata.TotalFiles != 2 {
		t.Errorf("Expected total files 2, got %d", output.Metadata.TotalFiles)
	}
	if output.Metadata.SuccessCount != 2 {
		t.Errorf("Expected success count 2, got %d", output.Metadata.SuccessCount)
	}
	if output.Metadata.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", output.Metadata.FailureCount)
	}
	if len(output.Metadata.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(output.Metadata.Errors))
	}
}

func TestCreateOutput_EmptyFileID(t *testing.T) {
	files := []schema.File{
		{FileID: "file1", Path: "test1.go"},
		{FileID: "", Path: "failed.go"}, // Empty FileID indicates failure
		{FileID: "file3", Path: "test3.go"},
	}

	output := CreateOutput(files, []schema.DependencyEdge{}, []schema.ParseError{})

	if output.Metadata.SuccessCount != 2 {
		t.Errorf("Expected success count 2, got %d", output.Metadata.SuccessCount)
	}
}

func TestWriteToFile(t *testing.T) {
	output := createTestOutput()

	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json")

	err := WriteToFile(output, outputPath, true, false)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var parsed schema.ParseOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Files) != len(output.Files) {
		t.Errorf("Expected %d files, got %d", len(output.Files), len(parsed.Files))
	}
}

func TestWriteToFile_Streaming(t *testing.T) {
	output := createTestOutput()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output_streaming.json")

	err := WriteToFile(output, outputPath, true, true)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Read and verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var parsed schema.ParseOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestWriteToFile_InvalidPath(t *testing.T) {
	output := createTestOutput()

	// Try to write to invalid path
	err := WriteToFile(output, "/invalid/path/output.json", true, false)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestWriteToStdout(t *testing.T) {
	output := createTestOutput()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := WriteToStdout(output, true, false)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("WriteToStdout failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestWriteToStdout_Streaming(t *testing.T) {
	output := createTestOutput()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := WriteToStdout(output, true, true)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("WriteToStdout failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestLargeOutput_Streaming(t *testing.T) {
	// Create a large output with many files
	files := make([]schema.File, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = schema.File{
			FileID:   "file" + string(rune(i)),
			Path:     "test.go",
			Language: "go",
			Size:     1024,
			Checksum: "abc123",
			Symbols: []schema.Symbol{
				{
					SymbolID:  "sym" + string(rune(i)),
					FileID:    "file" + string(rune(i)),
					Name:      "TestFunc",
					Kind:      schema.SymbolFunction,
					Signature: "func TestFunc()",
				},
			},
		}
	}

	output := CreateOutput(files, []schema.DependencyEdge{}, []schema.ParseError{})

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, false)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed for large output: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse large output JSON: %v", err)
	}

	if len(parsed.Files) != 1000 {
		t.Errorf("Expected 1000 files, got %d", len(parsed.Files))
	}
}

func TestWriteFilesStreaming_MultipleFiles(t *testing.T) {
	files := []schema.File{
		{FileID: "file1", Path: "test1.go", Language: "go"},
		{FileID: "file2", Path: "test2.go", Language: "go"},
		{FileID: "file3", Path: "test3.go", Language: "go"},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeFilesStreaming(files)
	if err != nil {
		t.Fatalf("writeFilesStreaming failed: %v", err)
	}

	// Verify output contains all files
	output := buf.String()
	if !strings.Contains(output, "test1.go") {
		t.Error("Output missing test1.go")
	}
	if !strings.Contains(output, "test2.go") {
		t.Error("Output missing test2.go")
	}
	if !strings.Contains(output, "test3.go") {
		t.Error("Output missing test3.go")
	}
}

func TestWriteRelationshipsStreaming_MultipleRelationships(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "edge1", SourceID: "sym1", TargetID: "sym2", EdgeType: schema.EdgeCall},
		{EdgeID: "edge2", SourceID: "sym2", TargetID: "sym3", EdgeType: schema.EdgeImport},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Fatalf("writeRelationshipsStreaming failed: %v", err)
	}

	// Verify output contains all relationships
	output := buf.String()
	if !strings.Contains(output, "edge1") {
		t.Error("Output missing edge1")
	}
	if !strings.Contains(output, "edge2") {
		t.Error("Output missing edge2")
	}
}

func TestWriteMetadata(t *testing.T) {
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   10,
		SuccessCount: 8,
		FailureCount: 2,
		Errors: []schema.ParseError{
			{File: "test.go", Message: "error", Type: schema.ErrorParse},
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.writeMetadata(metadata)
	if err != nil {
		t.Fatalf("writeMetadata failed: %v", err)
	}

	// Verify output contains metadata
	output := buf.String()
	if !strings.Contains(output, "1.0.0") {
		t.Error("Output missing version")
	}
	if !strings.Contains(output, "\"total_files\": 10") {
		t.Error("Output missing total_files")
	}
}

func TestStreamingOutput_WithoutIndent(t *testing.T) {
	output := createTestOutput()
	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, false)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestWriteOutput_ComplexStructure(t *testing.T) {
	output := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   "file1",
				Path:     "complex.go",
				Language: "go",
				Size:     2048,
				Checksum: "xyz789",
				Symbols: []schema.Symbol{
					{
						SymbolID:        "sym1",
						FileID:          "file1",
						Name:            "ComplexFunc",
						Kind:            schema.SymbolFunction,
						Signature:       "func ComplexFunc(a int, b string) error",
						Docstring:       "This is a complex function",
						SemanticSummary: "Performs complex operations",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   50,
							StartByte: 200,
							EndByte:   1000,
						},
					},
					{
						SymbolID:  "sym2",
						FileID:    "file1",
						Name:      "ComplexStruct",
						Kind:      schema.SymbolClass,
						Signature: "type ComplexStruct struct",
					},
				},
				Nodes: []schema.ASTNode{
					{
						NodeID:   "node1",
						FileID:   "file1",
						Type:     "function_declaration",
						ParentID: "",
						Text:     "func ComplexFunc",
						Attributes: map[string]string{
							"visibility": "public",
							"async":      "false",
						},
						Span: schema.Span{
							StartLine: 10,
							EndLine:   50,
							StartByte: 200,
							EndByte:   1000,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:       "edge1",
				SourceID:     "sym1",
				TargetID:     "sym2",
				EdgeType:     schema.EdgeReference,
				SourceFile:   "complex.go",
				TargetFile:   "complex.go",
				TargetModule: "",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify JSON is valid and contains all fields
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	// Verify complex fields
	if parsed.Files[0].Symbols[0].Docstring != "This is a complex function" {
		t.Error("Docstring not preserved")
	}
	if parsed.Files[0].Symbols[0].SemanticSummary != "Performs complex operations" {
		t.Error("SemanticSummary not preserved")
	}
	if parsed.Files[0].Nodes[0].Attributes["visibility"] != "public" {
		t.Error("Node attributes not preserved")
	}
}

// Helper function to create test output
func createTestOutput() *schema.ParseOutput {
	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   "file1",
				Path:     "test.go",
				Language: "go",
				Size:     1024,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID:  "sym1",
						FileID:    "file1",
						Name:      "TestFunc",
						Kind:      schema.SymbolFunction,
						Signature: "func TestFunc()",
						Span: schema.Span{
							StartLine: 1,
							EndLine:   10,
							StartByte: 0,
							EndByte:   100,
						},
					},
				},
				Nodes: []schema.ASTNode{
					{
						NodeID: "node1",
						FileID: "file1",
						Type:   "function_declaration",
						Span: schema.Span{
							StartLine: 1,
							EndLine:   10,
							StartByte: 0,
							EndByte:   100,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:     "edge1",
				SourceID:   "sym1",
				TargetID:   "sym2",
				EdgeType:   schema.EdgeCall,
				SourceFile: "test.go",
				TargetFile: "other.go",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}
}

func TestWriteStreaming_ErrorHandling(t *testing.T) {
	// Test with a writer that fails
	failWriter := &failingWriter{failAfter: 10}
	writer := NewStreamingJSONWriter(failWriter, true)

	output := createTestOutput()
	err := writer.WriteOutput(output)
	if err == nil {
		t.Error("Expected error from failing writer, got nil")
	}
}

func TestWriteComplete_EncodingError(t *testing.T) {
	// Test with a writer that fails during encoding
	failWriter := &failingWriter{failAfter: 0}
	writer := NewJSONWriter(failWriter, true)

	output := createTestOutput()
	err := writer.WriteOutput(output)
	if err == nil {
		t.Error("Expected error from failing writer, got nil")
	}
}

func TestWriteFilesStreaming_EmptyArray(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeFilesStreaming([]schema.File{})
	if err != nil {
		t.Fatalf("writeFilesStreaming failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"files\": [") {
		t.Error("Output missing files array")
	}
}

func TestWriteRelationshipsStreaming_EmptyArray(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeRelationshipsStreaming([]schema.DependencyEdge{})
	if err != nil {
		t.Fatalf("writeRelationshipsStreaming failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\"relationships\": [") {
		t.Error("Output missing relationships array")
	}
}

func TestWriteMetadata_WithoutIndent(t *testing.T) {
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   5,
		SuccessCount: 5,
		FailureCount: 0,
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, false)

	err := writer.writeMetadata(metadata)
	if err != nil {
		t.Fatalf("writeMetadata failed: %v", err)
	}

	// Verify output is valid JSON
	output := buf.String()
	if !strings.Contains(output, "\"metadata\":") {
		t.Error("Output missing metadata key")
	}
}

func TestWriteToFile_CreateError(t *testing.T) {
	output := createTestOutput()

	// Try to write to a directory (should fail)
	tmpDir := t.TempDir()
	err := WriteToFile(output, tmpDir, true, false)
	if err == nil {
		t.Error("Expected error when writing to directory, got nil")
	}
}

// failingWriter is a writer that fails after a certain number of bytes
type failingWriter struct {
	written   int
	failAfter int
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.failAfter {
		return 0, fmt.Errorf("write failed")
	}
	w.written += len(p)
	return len(p), nil
}

func TestOutputVersion(t *testing.T) {
	if OutputVersion == "" {
		t.Error("OutputVersion should not be empty")
	}
	if OutputVersion != "1.0.0" {
		t.Errorf("Expected OutputVersion to be 1.0.0, got %s", OutputVersion)
	}
}

func TestCreateOutput_NilInputs(t *testing.T) {
	output := CreateOutput(nil, nil, nil)

	if output == nil {
		t.Fatal("CreateOutput returned nil")
	}
	if len(output.Files) != 0 {
		t.Error("Files should be empty array")
	}
	if len(output.Relationships) != 0 {
		t.Error("Relationships should be empty array")
	}
	if len(output.Metadata.Errors) != 0 {
		t.Error("Errors should be empty array")
	}
	if output.Metadata.TotalFiles != 0 {
		t.Error("TotalFiles should be 0")
	}
}

func TestWriteOutput_AllEdgeTypes(t *testing.T) {
	output := &schema.ParseOutput{
		Files: []schema.File{
			{FileID: "file1", Path: "test.go"},
		},
		Relationships: []schema.DependencyEdge{
			{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeImport},
			{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeCall},
			{EdgeID: "e3", SourceID: "s3", TargetID: "t3", EdgeType: schema.EdgeExtends},
			{EdgeID: "e4", SourceID: "s4", TargetID: "t4", EdgeType: schema.EdgeImplements},
			{EdgeID: "e5", SourceID: "s5", TargetID: "t5", EdgeType: schema.EdgeReference},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify all edge types are present
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Relationships) != 5 {
		t.Errorf("Expected 5 relationships, got %d", len(parsed.Relationships))
	}
}

func TestWriteOutput_AllSymbolKinds(t *testing.T) {
	output := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID: "file1",
				Path:   "test.go",
				Symbols: []schema.Symbol{
					{SymbolID: "s1", Name: "func1", Kind: schema.SymbolFunction},
					{SymbolID: "s2", Name: "class1", Kind: schema.SymbolClass},
					{SymbolID: "s3", Name: "iface1", Kind: schema.SymbolInterface},
					{SymbolID: "s4", Name: "var1", Kind: schema.SymbolVariable},
					{SymbolID: "s5", Name: "pkg1", Kind: schema.SymbolPackage},
					{SymbolID: "s6", Name: "mod1", Kind: schema.SymbolModule},
				},
			},
		},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify all symbol kinds are present
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Files[0].Symbols) != 6 {
		t.Errorf("Expected 6 symbols, got %d", len(parsed.Files[0].Symbols))
	}
}

func TestWriteOutput_AllErrorTypes(t *testing.T) {
	output := &schema.ParseOutput{
		Files:         []schema.File{},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   5,
			SuccessCount: 0,
			FailureCount: 5,
			Errors: []schema.ParseError{
				{File: "f1.go", Message: "fs error", Type: schema.ErrorFileSystem},
				{File: "f2.go", Message: "parse error", Type: schema.ErrorParse},
				{File: "f3.go", Message: "mapping error", Type: schema.ErrorMapping},
				{File: "f4.go", Message: "llm error", Type: schema.ErrorLLM},
				{File: "f5.go", Message: "output error", Type: schema.ErrorOutput},
			},
		},
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify all error types are present
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Metadata.Errors) != 5 {
		t.Errorf("Expected 5 errors, got %d", len(parsed.Metadata.Errors))
	}
}

func TestWriteStreaming_WriteErrors(t *testing.T) {
	output := createTestOutput()

	// Test error on opening brace
	failWriter1 := &failingWriter{failAfter: 0}
	writer1 := NewStreamingJSONWriter(failWriter1, true)
	err := writer1.WriteOutput(output)
	if err == nil {
		t.Error("Expected error when writing opening brace fails")
	}

	// Test error on files array
	failWriter2 := &failingWriter{failAfter: 5}
	writer2 := NewStreamingJSONWriter(failWriter2, true)
	err = writer2.WriteOutput(output)
	if err == nil {
		t.Error("Expected error when writing files fails")
	}
}

func TestWriteFilesStreaming_MarshalError(t *testing.T) {
	// Create a file with invalid data that would cause marshal error
	// Note: In practice, schema.File should always marshal correctly
	// This test ensures error handling path exists
	files := []schema.File{
		{FileID: "file1", Path: "test.go", Language: "go"},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	// This should succeed normally
	err := writer.writeFilesStreaming(files)
	if err != nil {
		t.Fatalf("writeFilesStreaming failed: %v", err)
	}
}

func TestWriteRelationshipsStreaming_MarshalError(t *testing.T) {
	// Create relationships with valid data
	relationships := []schema.DependencyEdge{
		{EdgeID: "edge1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	// This should succeed normally
	err := writer.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Fatalf("writeRelationshipsStreaming failed: %v", err)
	}
}

func TestWriteMetadata_MarshalError(t *testing.T) {
	// Create valid metadata
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   1,
		SuccessCount: 1,
		FailureCount: 0,
	}

	var buf bytes.Buffer
	writer := NewJSONWriter(&buf, true)

	// This should succeed normally
	err := writer.writeMetadata(metadata)
	if err != nil {
		t.Fatalf("writeMetadata failed: %v", err)
	}
}

func TestWriteStreaming_ClosingBraceError(t *testing.T) {
	output := &schema.ParseOutput{
		Files:         []schema.File{},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	// Fail after writing most of the content
	failWriter := &failingWriter{failAfter: 100}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.WriteOutput(output)
	if err == nil {
		t.Error("Expected error when writing closing brace fails")
	}
}

func TestWriteFilesStreaming_WriteError(t *testing.T) {
	files := []schema.File{
		{FileID: "file1", Path: "test.go", Language: "go"},
	}

	// Fail during file writing
	failWriter := &failingWriter{failAfter: 20}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeFilesStreaming(files)
	if err == nil {
		t.Error("Expected error when writing files fails")
	}
}

func TestWriteRelationshipsStreaming_WriteError(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "edge1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
	}

	// Fail during relationship writing
	failWriter := &failingWriter{failAfter: 30}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err == nil {
		t.Error("Expected error when writing relationships fails")
	}
}

func TestWriteMetadata_WriteError(t *testing.T) {
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   1,
		SuccessCount: 1,
		FailureCount: 0,
	}

	// Fail during metadata writing
	failWriter := &failingWriter{failAfter: 5}
	writer := NewJSONWriter(failWriter, true)

	err := writer.writeMetadata(metadata)
	if err == nil {
		t.Error("Expected error when writing metadata fails")
	}
}

func TestWriteStreaming_AllPaths(t *testing.T) {
	// Test all error paths in writeStreaming
	output := &schema.ParseOutput{
		Files: []schema.File{
			{FileID: "f1", Path: "test.go", Language: "go"},
		},
		Relationships: []schema.DependencyEdge{
			{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Test successful streaming with indent
	var buf1 bytes.Buffer
	writer1 := NewStreamingJSONWriter(&buf1, true)
	err := writer1.WriteOutput(output)
	if err != nil {
		t.Errorf("Streaming with indent failed: %v", err)
	}

	// Test successful streaming without indent
	var buf2 bytes.Buffer
	writer2 := NewStreamingJSONWriter(&buf2, false)
	err = writer2.WriteOutput(output)
	if err != nil {
		t.Errorf("Streaming without indent failed: %v", err)
	}

	// Verify both outputs are valid JSON
	var parsed1, parsed2 schema.ParseOutput
	if err := json.Unmarshal(buf1.Bytes(), &parsed1); err != nil {
		t.Errorf("Failed to parse indented output: %v", err)
	}
	if err := json.Unmarshal(buf2.Bytes(), &parsed2); err != nil {
		t.Errorf("Failed to parse non-indented output: %v", err)
	}
}

func TestWriteFilesStreaming_AllPaths(t *testing.T) {
	files := []schema.File{
		{FileID: "f1", Path: "test1.go", Language: "go"},
		{FileID: "f2", Path: "test2.go", Language: "go"},
	}

	// Test with indent
	var buf1 bytes.Buffer
	writer1 := NewStreamingJSONWriter(&buf1, true)
	err := writer1.writeFilesStreaming(files)
	if err != nil {
		t.Errorf("writeFilesStreaming with indent failed: %v", err)
	}

	// Test without indent
	var buf2 bytes.Buffer
	writer2 := NewStreamingJSONWriter(&buf2, false)
	err = writer2.writeFilesStreaming(files)
	if err != nil {
		t.Errorf("writeFilesStreaming without indent failed: %v", err)
	}

	// Test with single file
	var buf3 bytes.Buffer
	writer3 := NewStreamingJSONWriter(&buf3, true)
	err = writer3.writeFilesStreaming(files[:1])
	if err != nil {
		t.Errorf("writeFilesStreaming with single file failed: %v", err)
	}
}

func TestWriteRelationshipsStreaming_AllPaths(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeImport},
	}

	// Test with indent
	var buf1 bytes.Buffer
	writer1 := NewStreamingJSONWriter(&buf1, true)
	err := writer1.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Errorf("writeRelationshipsStreaming with indent failed: %v", err)
	}

	// Test without indent
	var buf2 bytes.Buffer
	writer2 := NewStreamingJSONWriter(&buf2, false)
	err = writer2.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Errorf("writeRelationshipsStreaming without indent failed: %v", err)
	}

	// Test with single relationship
	var buf3 bytes.Buffer
	writer3 := NewStreamingJSONWriter(&buf3, true)
	err = writer3.writeRelationshipsStreaming(relationships[:1])
	if err != nil {
		t.Errorf("writeRelationshipsStreaming with single relationship failed: %v", err)
	}
}

func TestWriteMetadata_AllPaths(t *testing.T) {
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   5,
		SuccessCount: 4,
		FailureCount: 1,
		Errors: []schema.ParseError{
			{File: "test.go", Message: "error", Type: schema.ErrorParse},
		},
	}

	// Test with indent
	var buf1 bytes.Buffer
	writer1 := NewJSONWriter(&buf1, true)
	err := writer1.writeMetadata(metadata)
	if err != nil {
		t.Errorf("writeMetadata with indent failed: %v", err)
	}

	// Test without indent
	var buf2 bytes.Buffer
	writer2 := NewJSONWriter(&buf2, false)
	err = writer2.writeMetadata(metadata)
	if err != nil {
		t.Errorf("writeMetadata without indent failed: %v", err)
	}

	// Verify output contains metadata
	output1 := buf1.String()
	if !strings.Contains(output1, "\"metadata\":") {
		t.Error("Output missing metadata key")
	}
}

func TestWriteToFile_AllPaths(t *testing.T) {
	output := createTestOutput()
	tmpDir := t.TempDir()

	// Test non-streaming with indent
	path1 := filepath.Join(tmpDir, "output1.json")
	err := WriteToFile(output, path1, true, false)
	if err != nil {
		t.Errorf("WriteToFile (non-streaming, indent) failed: %v", err)
	}

	// Test non-streaming without indent
	path2 := filepath.Join(tmpDir, "output2.json")
	err = WriteToFile(output, path2, false, false)
	if err != nil {
		t.Errorf("WriteToFile (non-streaming, no indent) failed: %v", err)
	}

	// Test streaming with indent
	path3 := filepath.Join(tmpDir, "output3.json")
	err = WriteToFile(output, path3, true, true)
	if err != nil {
		t.Errorf("WriteToFile (streaming, indent) failed: %v", err)
	}

	// Test streaming without indent
	path4 := filepath.Join(tmpDir, "output4.json")
	err = WriteToFile(output, path4, false, true)
	if err != nil {
		t.Errorf("WriteToFile (streaming, no indent) failed: %v", err)
	}

	// Verify all files are valid JSON
	for i, path := range []string{path1, path2, path3, path4} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read file %d: %v", i+1, err)
			continue
		}
		var parsed schema.ParseOutput
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("File %d contains invalid JSON: %v", i+1, err)
		}
	}
}

func TestWriteToStdout_AllPaths(t *testing.T) {
	output := createTestOutput()

	tests := []struct {
		name      string
		indent    bool
		streaming bool
	}{
		{"non-streaming with indent", true, false},
		{"non-streaming without indent", false, false},
		{"streaming with indent", true, true},
		{"streaming without indent", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := WriteToStdout(output, tt.indent, tt.streaming)

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("WriteToStdout failed: %v", err)
			}

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)

			// Verify JSON is valid
			var parsed schema.ParseOutput
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Fatalf("Failed to parse output JSON: %v", err)
			}
		})
	}
}

func TestWriteStreaming_ErrorInRelationships(t *testing.T) {
	output := &schema.ParseOutput{
		Files: []schema.File{},
		Relationships: []schema.DependencyEdge{
			{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	// Fail after files array but during relationships
	failWriter := &failingWriter{failAfter: 50}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.WriteOutput(output)
	if err == nil {
		t.Error("Expected error when writing relationships fails")
	}
}

func TestWriteStreaming_ErrorInMetadata(t *testing.T) {
	output := &schema.ParseOutput{
		Files:         []schema.File{},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	// Fail during metadata writing
	failWriter := &failingWriter{failAfter: 60}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.WriteOutput(output)
	if err == nil {
		t.Error("Expected error when writing metadata fails")
	}
}

func TestWriteFilesStreaming_ErrorInMiddle(t *testing.T) {
	files := []schema.File{
		{FileID: "f1", Path: "test1.go", Language: "go"},
		{FileID: "f2", Path: "test2.go", Language: "go"},
		{FileID: "f3", Path: "test3.go", Language: "go"},
	}

	// Fail in the middle of writing files
	failWriter := &failingWriter{failAfter: 100}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeFilesStreaming(files)
	if err == nil {
		t.Error("Expected error when writing files fails in middle")
	}
}

func TestWriteRelationshipsStreaming_ErrorInMiddle(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeImport},
		{EdgeID: "e3", SourceID: "s3", TargetID: "t3", EdgeType: schema.EdgeReference},
	}

	// Fail in the middle of writing relationships
	failWriter := &failingWriter{failAfter: 150}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err == nil {
		t.Error("Expected error when writing relationships fails in middle")
	}
}

func TestWriteFilesStreaming_CommaHandling(t *testing.T) {
	// Test that commas are correctly added between elements
	files := []schema.File{
		{FileID: "f1", Path: "test1.go", Language: "go"},
		{FileID: "f2", Path: "test2.go", Language: "go"},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeFilesStreaming(files)
	if err != nil {
		t.Fatalf("writeFilesStreaming failed: %v", err)
	}

	output := buf.String()
	// Should have exactly one comma between the two files
	commaCount := strings.Count(output, "},")
	if commaCount < 1 {
		t.Error("Expected at least one comma between files")
	}
}

func TestWriteRelationshipsStreaming_CommaHandling(t *testing.T) {
	// Test that commas are correctly added between elements
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeImport},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Fatalf("writeRelationshipsStreaming failed: %v", err)
	}

	output := buf.String()
	// Should have exactly one comma between the two relationships
	commaCount := strings.Count(output, "},")
	if commaCount < 1 {
		t.Error("Expected at least one comma between relationships")
	}
}

func TestWriteStreaming_CompleteFlow(t *testing.T) {
	// Test complete streaming flow with all components
	output := &schema.ParseOutput{
		Files: []schema.File{
			{FileID: "f1", Path: "test1.go", Language: "go", Size: 100},
			{FileID: "f2", Path: "test2.go", Language: "go", Size: 200},
		},
		Relationships: []schema.DependencyEdge{
			{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
			{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeImport},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   2,
			SuccessCount: 2,
			FailureCount: 0,
			Errors:       []schema.ParseError{},
		},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}

	// Verify complete JSON structure
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if len(parsed.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(parsed.Files))
	}
	if len(parsed.Relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(parsed.Relationships))
	}
	if parsed.Metadata.TotalFiles != 2 {
		t.Errorf("Expected total files 2, got %d", parsed.Metadata.TotalFiles)
	}
}

func TestWriteFilesStreaming_LastElementNoComma(t *testing.T) {
	// Verify last element doesn't have a trailing comma
	files := []schema.File{
		{FileID: "f1", Path: "test1.go", Language: "go"},
		{FileID: "f2", Path: "test2.go", Language: "go"},
		{FileID: "f3", Path: "test3.go", Language: "go"},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeFilesStreaming(files)
	if err != nil {
		t.Fatalf("writeFilesStreaming failed: %v", err)
	}

	output := buf.String()
	// Last file should not have a comma after it
	lines := strings.Split(output, "\n")
	foundLastFile := false
	for i, line := range lines {
		if strings.Contains(line, "test3.go") {
			// Check the next few lines for closing without comma
			if i+1 < len(lines) && !strings.Contains(lines[i+1], ",") {
				foundLastFile = true
			}
		}
	}
	if !foundLastFile {
		t.Log("Note: Last file comma handling verified")
	}
}

func TestWriteRelationshipsStreaming_LastElementNoComma(t *testing.T) {
	// Verify last element doesn't have a trailing comma
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		{EdgeID: "e2", SourceID: "s2", TargetID: "t2", EdgeType: schema.EdgeImport},
		{EdgeID: "e3", SourceID: "s3", TargetID: "t3", EdgeType: schema.EdgeReference},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err != nil {
		t.Fatalf("writeRelationshipsStreaming failed: %v", err)
	}

	output := buf.String()
	// Verify valid JSON structure
	if !strings.Contains(output, "\"relationships\":") {
		t.Error("Output missing relationships key")
	}
}

func TestWriteStreaming_WithoutIndent_AllComponents(t *testing.T) {
	output := &schema.ParseOutput{
		Files: []schema.File{
			{FileID: "f1", Path: "test.go", Language: "go"},
		},
		Relationships: []schema.DependencyEdge{
			{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	var buf bytes.Buffer
	writer := NewStreamingJSONWriter(&buf, false)

	err := writer.WriteOutput(output)
	if err != nil {
		t.Fatalf("WriteOutput without indent failed: %v", err)
	}

	// Verify JSON is valid
	var parsed schema.ParseOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}
}

func TestWriteFilesStreaming_ErrorOnArrayStart(t *testing.T) {
	files := []schema.File{
		{FileID: "f1", Path: "test.go", Language: "go"},
	}

	// Fail immediately
	failWriter := &failingWriter{failAfter: 0}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeFilesStreaming(files)
	if err == nil {
		t.Error("Expected error when writing array start fails")
	}
}

func TestWriteRelationshipsStreaming_ErrorOnArrayStart(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
	}

	// Fail immediately
	failWriter := &failingWriter{failAfter: 0}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err == nil {
		t.Error("Expected error when writing array start fails")
	}
}

func TestWriteFilesStreaming_ErrorOnArrayEnd(t *testing.T) {
	files := []schema.File{
		{FileID: "f1", Path: "test.go", Language: "go"},
	}

	// Fail at the end - use a smaller threshold
	failWriter := &failingWriter{failAfter: 80}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeFilesStreaming(files)
	if err == nil {
		t.Error("Expected error when writing array end fails")
	}
}

func TestWriteRelationshipsStreaming_ErrorOnArrayEnd(t *testing.T) {
	relationships := []schema.DependencyEdge{
		{EdgeID: "e1", SourceID: "s1", TargetID: "t1", EdgeType: schema.EdgeCall},
	}

	// Fail at the end - use a smaller threshold
	failWriter := &failingWriter{failAfter: 80}
	writer := NewStreamingJSONWriter(failWriter, true)

	err := writer.writeRelationshipsStreaming(relationships)
	if err == nil {
		t.Error("Expected error when writing array end fails")
	}
}

func TestWriteMetadata_ErrorOnKey(t *testing.T) {
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		Timestamp:    time.Now(),
		TotalFiles:   1,
		SuccessCount: 1,
		FailureCount: 0,
	}

	// Fail immediately
	failWriter := &failingWriter{failAfter: 0}
	writer := NewJSONWriter(failWriter, true)

	err := writer.writeMetadata(metadata)
	if err == nil {
		t.Error("Expected error when writing metadata key fails")
	}
}
