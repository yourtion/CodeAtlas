package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

const (
	// Version of the parse output format
	OutputVersion = "1.0.0"
)

// JSONWriter handles JSON serialization and output
type JSONWriter struct {
	writer     io.Writer
	indent     bool
	streaming  bool
	bufferSize int
}

// NewJSONWriter creates a new JSON writer
func NewJSONWriter(writer io.Writer, indent bool) *JSONWriter {
	return &JSONWriter{
		writer:     writer,
		indent:     indent,
		streaming:  false,
		bufferSize: 100, // Buffer size for streaming mode
	}
}

// NewStreamingJSONWriter creates a JSON writer optimized for large outputs
func NewStreamingJSONWriter(writer io.Writer, indent bool) *JSONWriter {
	return &JSONWriter{
		writer:     writer,
		indent:     indent,
		streaming:  true,
		bufferSize: 100,
	}
}

// WriteOutput writes the complete parse output as JSON
func (w *JSONWriter) WriteOutput(output *schema.ParseOutput) error {
	if w.streaming {
		return w.writeStreaming(output)
	}
	return w.writeComplete(output)
}

// writeComplete writes the entire output at once (for smaller outputs)
func (w *JSONWriter) writeComplete(output *schema.ParseOutput) error {
	encoder := json.NewEncoder(w.writer)
	if w.indent {
		encoder.SetIndent("", "  ")
	}
	
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	
	return nil
}

// writeStreaming writes output in streaming fashion (for large repositories)
func (w *JSONWriter) writeStreaming(output *schema.ParseOutput) error {
	// Write opening brace
	if _, err := w.writer.Write([]byte("{\n")); err != nil {
		return err
	}

	// Write files array
	if err := w.writeFilesStreaming(output.Files); err != nil {
		return err
	}

	// Write relationships array
	if err := w.writeRelationshipsStreaming(output.Relationships); err != nil {
		return err
	}

	// Write metadata
	if err := w.writeMetadata(output.Metadata); err != nil {
		return err
	}

	// Write closing brace
	if _, err := w.writer.Write([]byte("\n}\n")); err != nil {
		return err
	}

	return nil
}

// writeFilesStreaming writes files array in streaming fashion
func (w *JSONWriter) writeFilesStreaming(files []schema.File) error {
	indent := ""
	if w.indent {
		indent = "  "
	}

	if _, err := w.writer.Write([]byte(indent + "\"files\": [\n")); err != nil {
		return err
	}

	for i, file := range files {
		fileJSON, err := json.MarshalIndent(file, indent+"  ", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal file: %w", err)
		}

		if _, err := w.writer.Write([]byte(indent + "  ")); err != nil {
			return err
		}
		if _, err := w.writer.Write(fileJSON); err != nil {
			return err
		}

		// Add comma if not last element
		if i < len(files)-1 {
			if _, err := w.writer.Write([]byte(",")); err != nil {
				return err
			}
		}
		if _, err := w.writer.Write([]byte("\n")); err != nil {
			return err
		}
	}

	if _, err := w.writer.Write([]byte(indent + "],\n")); err != nil {
		return err
	}

	return nil
}

// writeRelationshipsStreaming writes relationships array in streaming fashion
func (w *JSONWriter) writeRelationshipsStreaming(relationships []schema.DependencyEdge) error {
	indent := ""
	if w.indent {
		indent = "  "
	}

	if _, err := w.writer.Write([]byte(indent + "\"relationships\": [\n")); err != nil {
		return err
	}

	for i, rel := range relationships {
		relJSON, err := json.MarshalIndent(rel, indent+"  ", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal relationship: %w", err)
		}

		if _, err := w.writer.Write([]byte(indent + "  ")); err != nil {
			return err
		}
		if _, err := w.writer.Write(relJSON); err != nil {
			return err
		}

		// Add comma if not last element
		if i < len(relationships)-1 {
			if _, err := w.writer.Write([]byte(",")); err != nil {
				return err
			}
		}
		if _, err := w.writer.Write([]byte("\n")); err != nil {
			return err
		}
	}

	if _, err := w.writer.Write([]byte(indent + "],\n")); err != nil {
		return err
	}

	return nil
}

// writeMetadata writes metadata section
func (w *JSONWriter) writeMetadata(metadata schema.ParseMetadata) error {
	indent := ""
	if w.indent {
		indent = "  "
	}

	metadataJSON, err := json.MarshalIndent(metadata, indent, "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if _, err := w.writer.Write([]byte(indent + "\"metadata\": ")); err != nil {
		return err
	}
	if _, err := w.writer.Write(metadataJSON); err != nil {
		return err
	}

	return nil
}

// CreateOutput creates a ParseOutput structure with metadata
func CreateOutput(files []schema.File, relationships []schema.DependencyEdge, errors []schema.ParseError) *schema.ParseOutput {
	// Handle nil inputs
	if files == nil {
		files = []schema.File{}
	}
	if relationships == nil {
		relationships = []schema.DependencyEdge{}
	}
	if errors == nil {
		errors = []schema.ParseError{}
	}

	successCount := 0
	for _, file := range files {
		if file.FileID != "" {
			successCount++
		}
	}

	metadata := schema.ParseMetadata{
		Version:      OutputVersion,
		Timestamp:    time.Now(),
		TotalFiles:   len(files),
		SuccessCount: successCount,
		FailureCount: len(errors),
		Errors:       errors,
	}

	return &schema.ParseOutput{
		Files:         files,
		Relationships: relationships,
		Metadata:      metadata,
	}
}

// WriteToFile writes output to a file
func WriteToFile(output *schema.ParseOutput, filepath string, indent bool, streaming bool) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	var writer *JSONWriter
	if streaming {
		writer = NewStreamingJSONWriter(file, indent)
	} else {
		writer = NewJSONWriter(file, indent)
	}

	if err := writer.WriteOutput(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// WriteToStdout writes output to stdout
func WriteToStdout(output *schema.ParseOutput, indent bool, streaming bool) error {
	var writer *JSONWriter
	if streaming {
		writer = NewStreamingJSONWriter(os.Stdout, indent)
	} else {
		writer = NewJSONWriter(os.Stdout, indent)
	}

	if err := writer.WriteOutput(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}
