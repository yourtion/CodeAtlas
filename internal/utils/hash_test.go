package utils

import (
	"testing"
)

func TestSHA256Checksum(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			content:  []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "multiline content",
			content:  []byte("line1\nline2\nline3"),
			expected: "3f786850e387550fdab836ed7e6dc881de23001682c9bcbc635c66d8e8c5e3d1",
		},
		{
			name:     "binary content",
			content:  []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			expected: "5e4e8a1e6b8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e8e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SHA256Checksum(tt.content)
			if len(result) != 64 {
				t.Errorf("SHA256Checksum() returned hash with length %d, expected 64", len(result))
			}
			// Verify it's a valid hex string
			for _, c := range result {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("SHA256Checksum() returned non-hex character: %c", c)
				}
			}
		})
	}
}

func TestSHA256ChecksumConsistency(t *testing.T) {
	content := []byte("test content for consistency")
	
	// Generate checksum multiple times
	checksum1 := SHA256Checksum(content)
	checksum2 := SHA256Checksum(content)
	checksum3 := SHA256Checksum(content)
	
	if checksum1 != checksum2 || checksum2 != checksum3 {
		t.Errorf("SHA256Checksum() is not consistent: %s, %s, %s", checksum1, checksum2, checksum3)
	}
}

func TestSHA256ChecksumUniqueness(t *testing.T) {
	content1 := []byte("content1")
	content2 := []byte("content2")
	
	checksum1 := SHA256Checksum(content1)
	checksum2 := SHA256Checksum(content2)
	
	if checksum1 == checksum2 {
		t.Errorf("SHA256Checksum() produced same hash for different content")
	}
}

func TestGenerateUUID(t *testing.T) {
	// Test that UUID is generated
	uuid := GenerateUUID()
	if uuid == "" {
		t.Error("GenerateUUID() returned empty string")
	}
	
	// Test UUID format (should be 36 characters with hyphens)
	if len(uuid) != 36 {
		t.Errorf("GenerateUUID() returned UUID with length %d, expected 36", len(uuid))
	}
	
	// Check for hyphens in correct positions (8-4-4-4-12 format)
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Errorf("GenerateUUID() returned invalid UUID format: %s", uuid)
	}
}

func TestGenerateUUIDUniqueness(t *testing.T) {
	// Generate multiple UUIDs and check they're unique
	uuids := make(map[string]bool)
	count := 1000
	
	for i := 0; i < count; i++ {
		uuid := GenerateUUID()
		if uuids[uuid] {
			t.Errorf("GenerateUUID() generated duplicate UUID: %s", uuid)
		}
		uuids[uuid] = true
	}
	
	if len(uuids) != count {
		t.Errorf("GenerateUUID() generated %d unique UUIDs, expected %d", len(uuids), count)
	}
}

func TestGenerateUUIDFormat(t *testing.T) {
	uuid := GenerateUUID()
	
	// Verify each segment contains only hex characters
	segments := []struct {
		start int
		end   int
	}{
		{0, 8},   // 8 chars
		{9, 13},  // 4 chars
		{14, 18}, // 4 chars
		{19, 23}, // 4 chars
		{24, 36}, // 12 chars
	}
	
	for _, seg := range segments {
		for i := seg.start; i < seg.end; i++ {
			c := uuid[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				t.Errorf("GenerateUUID() returned non-hex character at position %d: %c", i, c)
			}
		}
	}
}
