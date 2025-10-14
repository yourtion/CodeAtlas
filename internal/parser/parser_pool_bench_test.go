package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// BenchmarkParserPool benchmarks the parser pool with different worker counts
func BenchmarkParserPool(b *testing.B) {
	// Create test files
	testDir := b.TempDir()
	files := createBenchmarkFiles(b, testDir, 50)

	// Initialize Tree-sitter parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	workerCounts := []int{1, 2, 4, 8, runtime.NumCPU()}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool := NewParserPool(workers, tsParser)
				_, _ = pool.Process(files)
			}
		})
	}
}

// BenchmarkParserPoolLarge benchmarks with a larger number of files
func BenchmarkParserPoolLarge(b *testing.B) {
	testDir := b.TempDir()
	files := createBenchmarkFiles(b, testDir, 200)

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := NewParserPool(runtime.NumCPU(), tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.Process(files)
	}
}

// BenchmarkGoParser benchmarks Go file parsing
func BenchmarkGoParser(b *testing.B) {
	testDir := b.TempDir()
	goFile := createGoFile(b, testDir, "test.go")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewGoParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(goFile)
	}
}

// BenchmarkJSParser benchmarks JavaScript file parsing
func BenchmarkJSParser(b *testing.B) {
	testDir := b.TempDir()
	jsFile := createJSFile(b, testDir, "test.js")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewJSParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(jsFile)
	}
}

// BenchmarkPythonParser benchmarks Python file parsing
func BenchmarkPythonParser(b *testing.B) {
	testDir := b.TempDir()
	pyFile := createPythonFile(b, testDir, "test.py")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(pyFile)
	}
}

// BenchmarkFileScanner benchmarks directory scanning
func BenchmarkFileScanner(b *testing.B) {
	testDir := b.TempDir()
	createBenchmarkFiles(b, testDir, 100)

	filter, err := NewIgnoreFilter(nil, nil)
	if err != nil {
		b.Fatalf("Failed to create ignore filter: %v", err)
	}

	scanner := NewFileScanner(testDir, filter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.Scan()
	}
}

// BenchmarkIgnoreFilter benchmarks ignore filter performance
func BenchmarkIgnoreFilter(b *testing.B) {
	patterns := []string{
		"*.test.js",
		"node_modules/**",
		"vendor/**",
		"*.pyc",
		"__pycache__/**",
	}

	filter, err := NewIgnoreFilter(nil, patterns)
	if err != nil {
		b.Fatalf("Failed to create ignore filter: %v", err)
	}

	testPaths := []string{
		"src/main.go",
		"src/test.test.js",
		"node_modules/package/index.js",
		"vendor/lib/file.go",
		"src/__pycache__/module.pyc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			filter.ShouldIgnore(path, false)
		}
	}
}

// Helper functions to create benchmark files

func createBenchmarkFiles(b *testing.B, dir string, count int) []ScannedFile {
	var files []ScannedFile

	// Create a mix of Go, JS, and Python files
	for i := 0; i < count; i++ {
		var file ScannedFile
		switch i % 3 {
		case 0:
			file = createGoFile(b, dir, fmt.Sprintf("file%d.go", i))
		case 1:
			file = createJSFile(b, dir, fmt.Sprintf("file%d.js", i))
		case 2:
			file = createPythonFile(b, dir, fmt.Sprintf("file%d.py", i))
		}
		files = append(files, file)
	}

	return files
}

func createGoFile(b *testing.B, dir, name string) ScannedFile {
	content := `package main

import (
	"fmt"
	"os"
)

// User represents a user in the system
type User struct {
	ID   int
	Name string
	Email string
}

// NewUser creates a new user
func NewUser(id int, name, email string) *User {
	return &User{
		ID:   id,
		Name: name,
		Email: email,
	}
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}

// SetName sets the user's name
func (u *User) SetName(name string) {
	u.Name = name
}

func main() {
	user := NewUser(1, "John Doe", "john@example.com")
	fmt.Println(user.GetName())
}
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create Go file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Go",
		Size:     info.Size(),
	}
}

func createJSFile(b *testing.B, dir, name string) ScannedFile {
	content := `/**
 * User class representing a user in the system
 */
class User {
	constructor(id, name, email) {
		this.id = id;
		this.name = name;
		this.email = email;
	}

	/**
	 * Get the user's name
	 * @returns {string} The user's name
	 */
	getName() {
		return this.name;
	}

	/**
	 * Set the user's name
	 * @param {string} name - The new name
	 */
	setName(name) {
		this.name = name;
	}
}

/**
 * Create a new user
 * @param {number} id - User ID
 * @param {string} name - User name
 * @param {string} email - User email
 * @returns {User} New user instance
 */
function createUser(id, name, email) {
	return new User(id, name, email);
}

const user = createUser(1, "John Doe", "john@example.com");
console.log(user.getName());
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create JS file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "JavaScript",
		Size:     info.Size(),
	}
}

func createPythonFile(b *testing.B, dir, name string) ScannedFile {
	content := `"""
User module for managing user data
"""

class User:
	"""Represents a user in the system"""
	
	def __init__(self, id: int, name: str, email: str):
		"""
		Initialize a new user
		
		Args:
			id: User ID
			name: User name
			email: User email
		"""
		self.id = id
		self.name = name
		self.email = email
	
	def get_name(self) -> str:
		"""Get the user's name"""
		return self.name
	
	def set_name(self, name: str) -> None:
		"""Set the user's name"""
		self.name = name

def create_user(id: int, name: str, email: str) -> User:
	"""
	Create a new user
	
	Args:
		id: User ID
		name: User name
		email: User email
	
	Returns:
		New User instance
	"""
	return User(id, name, email)

if __name__ == "__main__":
	user = create_user(1, "John Doe", "john@example.com")
	print(user.get_name())
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create Python file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Python",
		Size:     info.Size(),
	}
}
