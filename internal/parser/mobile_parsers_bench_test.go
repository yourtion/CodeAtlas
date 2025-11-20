package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// BenchmarkKotlinParser benchmarks Kotlin file parsing
func BenchmarkKotlinParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	kotlinFile := createKotlinFile(b, testDir, "test.kt")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(kotlinFile)
	}
}

// BenchmarkJavaParser benchmarks Java file parsing
func BenchmarkJavaParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	javaFile := createJavaFile(b, testDir, "test.java")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(javaFile)
	}
}

// BenchmarkSwiftParser benchmarks Swift file parsing
func BenchmarkSwiftParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	swiftFile := createSwiftFile(b, testDir, "test.swift")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewSwiftParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(swiftFile)
	}
}

// BenchmarkObjCParser benchmarks Objective-C file parsing
func BenchmarkObjCParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	objcFile := createObjCFile(b, testDir, "test.m")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewObjCParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(objcFile)
	}
}

// BenchmarkCParser benchmarks C file parsing
func BenchmarkCParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	cFile := createCFile(b, testDir, "test.c")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewCParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(cFile)
	}
}

// BenchmarkCppParser benchmarks C++ file parsing
func BenchmarkCppParser(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	cppFile := createCppFile(b, testDir, "test.cpp")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewCppParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(cppFile)
	}
}

// BenchmarkAllParsers compares performance across all parsers
func BenchmarkAllParsers(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parsers := map[string]struct {
		file   ScannedFile
		parser interface{ Parse(ScannedFile) (*ParsedFile, error) }
	}{
		"Go":         {createGoFile(b, testDir, "test.go"), NewGoParser(tsParser)},
		"JavaScript": {createJSFile(b, testDir, "test.js"), NewJSParser(tsParser)},
		"Python":     {createPythonFile(b, testDir, "test.py"), NewPythonParser(tsParser)},
		"Kotlin":     {createKotlinFile(b, testDir, "test.kt"), NewKotlinParser(tsParser)},
		"Java":       {createJavaFile(b, testDir, "test.java"), NewJavaParser(tsParser)},
		"Swift":      {createSwiftFile(b, testDir, "test.swift"), NewSwiftParser(tsParser)},
		"ObjC":       {createObjCFile(b, testDir, "test.m"), NewObjCParser(tsParser)},
		"C":          {createCFile(b, testDir, "test.c"), NewCParser(tsParser)},
		"Cpp":        {createCppFile(b, testDir, "test.cpp"), NewCppParser(tsParser)},
	}

	for name, p := range parsers {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = p.parser.Parse(p.file)
			}
		})
	}
}

// BenchmarkMobileParserPool benchmarks parser pool with mobile languages
func BenchmarkMobileParserPool(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	files := createMobileBenchmarkFiles(b, testDir, 100)

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

// BenchmarkLargeKotlinFile benchmarks parsing a large Kotlin file
func BenchmarkLargeKotlinFile(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	kotlinFile := createLargeKotlinFile(b, testDir, "large.kt")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(kotlinFile)
	}
}

// BenchmarkLargeJavaFile benchmarks parsing a large Java file
func BenchmarkLargeJavaFile(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	javaFile := createLargeJavaFile(b, testDir, "large.java")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(javaFile)
	}
}

// BenchmarkLargeSwiftFile benchmarks parsing a large Swift file
func BenchmarkLargeSwiftFile(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	swiftFile := createLargeSwiftFile(b, testDir, "large.swift")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewSwiftParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(swiftFile)
	}
}

// BenchmarkLargeCppFile benchmarks parsing a large C++ file
func BenchmarkLargeCppFile(b *testing.B) {
	// Keep b for benchmark functions
	testDir := b.TempDir()
	cppFile := createLargeCppFile(b, testDir, "large.cpp")

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		b.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewCppParser(tsParser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(cppFile)
	}
}

// Helper functions to create benchmark files

func createMobileBenchmarkFiles(tb testingTB, dir string, count int) []ScannedFile {
	var files []ScannedFile

	// Create a mix of mobile language files
	for i := 0; i < count; i++ {
		var file ScannedFile
		switch i % 6 {
		case 0:
			file = createKotlinFile(tb, dir, fmt.Sprintf("file%d.kt", i))
		case 1:
			file = createJavaFile(tb, dir, fmt.Sprintf("file%d.java", i))
		case 2:
			file = createSwiftFile(tb, dir, fmt.Sprintf("file%d.swift", i))
		case 3:
			file = createObjCFile(tb, dir, fmt.Sprintf("file%d.m", i))
		case 4:
			file = createCFile(tb, dir, fmt.Sprintf("file%d.c", i))
		case 5:
			file = createCppFile(tb, dir, fmt.Sprintf("file%d.cpp", i))
		}
		files = append(files, file)
	}

	return files
}

// testingTB is an interface that both *testing.T and *testing.B implement
type testingTB interface {
	TempDir() string
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

func createKotlinFile(tb testingTB, dir, name string) ScannedFile {
	content := `package com.example.app

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity

/**
 * User data class representing a user in the system
 */
data class User(
    val id: Int,
    val name: String,
    val email: String
)

/**
 * User repository for managing user data
 */
class UserRepository {
    private val users = mutableListOf<User>()

    /**
     * Add a new user to the repository
     */
    fun addUser(user: User) {
        users.add(user)
    }

    /**
     * Get user by ID
     */
    fun getUserById(id: Int): User? {
        return users.find { it.id == id }
    }

    /**
     * Get all users
     */
    fun getAllUsers(): List<User> {
        return users.toList()
    }
}

/**
 * Main activity
 */
class MainActivity : AppCompatActivity() {
    private val repository = UserRepository()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        val user = User(1, "John Doe", "john@example.com")
        repository.addUser(user)
    }
}
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create Kotlin file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Kotlin",
		Size:     info.Size(),
	}
}

func createJavaFile(tb testingTB, dir, name string) ScannedFile {
	content := `package com.example.app;

import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

/**
 * User class representing a user in the system
 */
public class User {
    private int id;
    private String name;
    private String email;

    /**
     * Constructor for User
     */
    public User(int id, String name, String email) {
        this.id = id;
        this.name = name;
        this.email = email;
    }

    /**
     * Get user ID
     */
    public int getId() {
        return id;
    }

    /**
     * Get user name
     */
    public String getName() {
        return name;
    }

    /**
     * Set user name
     */
    public void setName(String name) {
        this.name = name;
    }

    /**
     * Get user email
     */
    public String getEmail() {
        return email;
    }
}

/**
 * User repository for managing user data
 */
public class UserRepository {
    private List<User> users = new ArrayList<>();

    /**
     * Add a new user
     */
    public void addUser(User user) {
        users.add(user);
    }

    /**
     * Get user by ID
     */
    public Optional<User> getUserById(int id) {
        return users.stream()
            .filter(u -> u.getId() == id)
            .findFirst();
    }

    /**
     * Get all users
     */
    public List<User> getAllUsers() {
        return new ArrayList<>(users);
    }
}
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create Java file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Java",
		Size:     info.Size(),
	}
}

func createSwiftFile(tb testingTB, dir, name string) ScannedFile {
	content := `import Foundation
import UIKit

/// User struct representing a user in the system
struct User {
    let id: Int
    var name: String
    var email: String
    
    /// Initialize a new user
    init(id: Int, name: String, email: String) {
        self.id = id
        self.name = name
        self.email = email
    }
}

/// User repository protocol
protocol UserRepositoryProtocol {
    func addUser(_ user: User)
    func getUserById(_ id: Int) -> User?
    func getAllUsers() -> [User]
}

/// User repository implementation
class UserRepository: UserRepositoryProtocol {
    private var users: [User] = []
    
    /// Add a new user to the repository
    func addUser(_ user: User) {
        users.append(user)
    }
    
    /// Get user by ID
    func getUserById(_ id: Int) -> User? {
        return users.first { $0.id == id }
    }
    
    /// Get all users
    func getAllUsers() -> [User] {
        return users
    }
}

/// Main view controller
class ViewController: UIViewController {
    private let repository = UserRepository()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        
        let user = User(id: 1, name: "John Doe", email: "john@example.com")
        repository.addUser(user)
    }
}
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create Swift file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Swift",
		Size:     info.Size(),
	}
}

func createObjCFile(tb testingTB, dir, name string) ScannedFile {
	content := `#import <Foundation/Foundation.h>
#import <UIKit/UIKit.h>

/**
 * User interface
 */
@interface User : NSObject

@property (nonatomic, assign) NSInteger userId;
@property (nonatomic, strong) NSString *name;
@property (nonatomic, strong) NSString *email;

- (instancetype)initWithId:(NSInteger)userId 
                      name:(NSString *)name 
                     email:(NSString *)email;
- (NSString *)getName;
- (void)setName:(NSString *)name;

@end

/**
 * User implementation
 */
@implementation User

- (instancetype)initWithId:(NSInteger)userId 
                      name:(NSString *)name 
                     email:(NSString *)email {
    self = [super init];
    if (self) {
        _userId = userId;
        _name = name;
        _email = email;
    }
    return self;
}

- (NSString *)getName {
    return self.name;
}

- (void)setName:(NSString *)name {
    self.name = name;
}

@end

/**
 * User repository interface
 */
@interface UserRepository : NSObject

- (void)addUser:(User *)user;
- (User *)getUserById:(NSInteger)userId;
- (NSArray<User *> *)getAllUsers;

@end

/**
 * User repository implementation
 */
@implementation UserRepository {
    NSMutableArray<User *> *_users;
}

- (instancetype)init {
    self = [super init];
    if (self) {
        _users = [[NSMutableArray alloc] init];
    }
    return self;
}

- (void)addUser:(User *)user {
    [_users addObject:user];
}

- (User *)getUserById:(NSInteger)userId {
    for (User *user in _users) {
        if (user.userId == userId) {
            return user;
        }
    }
    return nil;
}

- (NSArray<User *> *)getAllUsers {
    return [_users copy];
}

@end
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create Objective-C file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Objective-C",
		Size:     info.Size(),
	}
}

func createCFile(tb testingTB, dir, name string) ScannedFile {
	content := `#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/**
 * User structure
 */
typedef struct {
    int id;
    char name[100];
    char email[100];
} User;

/**
 * Create a new user
 */
User* create_user(int id, const char* name, const char* email) {
    User* user = (User*)malloc(sizeof(User));
    if (user == NULL) {
        return NULL;
    }
    
    user->id = id;
    strncpy(user->name, name, sizeof(user->name) - 1);
    strncpy(user->email, email, sizeof(user->email) - 1);
    
    return user;
}

/**
 * Get user name
 */
const char* get_user_name(const User* user) {
    return user->name;
}

/**
 * Set user name
 */
void set_user_name(User* user, const char* name) {
    strncpy(user->name, name, sizeof(user->name) - 1);
}

/**
 * Free user memory
 */
void free_user(User* user) {
    free(user);
}

/**
 * User repository structure
 */
typedef struct {
    User** users;
    int count;
    int capacity;
} UserRepository;

/**
 * Create a new repository
 */
UserRepository* create_repository(int capacity) {
    UserRepository* repo = (UserRepository*)malloc(sizeof(UserRepository));
    if (repo == NULL) {
        return NULL;
    }
    
    repo->users = (User**)malloc(sizeof(User*) * capacity);
    repo->count = 0;
    repo->capacity = capacity;
    
    return repo;
}

/**
 * Add user to repository
 */
int add_user(UserRepository* repo, User* user) {
    if (repo->count >= repo->capacity) {
        return -1;
    }
    
    repo->users[repo->count++] = user;
    return 0;
}

/**
 * Get user by ID
 */
User* get_user_by_id(const UserRepository* repo, int id) {
    for (int i = 0; i < repo->count; i++) {
        if (repo->users[i]->id == id) {
            return repo->users[i];
        }
    }
    return NULL;
}

/**
 * Free repository memory
 */
void free_repository(UserRepository* repo) {
    for (int i = 0; i < repo->count; i++) {
        free_user(repo->users[i]);
    }
    free(repo->users);
    free(repo);
}
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create C file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "C",
		Size:     info.Size(),
	}
}

func createCppFile(tb testingTB, dir, name string) ScannedFile {
	content := `#include <iostream>
#include <string>
#include <vector>
#include <memory>
#include <algorithm>

namespace app {

/**
 * User class representing a user in the system
 */
class User {
private:
    int id_;
    std::string name_;
    std::string email_;

public:
    /**
     * Constructor
     */
    User(int id, const std::string& name, const std::string& email)
        : id_(id), name_(name), email_(email) {}

    /**
     * Get user ID
     */
    int getId() const {
        return id_;
    }

    /**
     * Get user name
     */
    const std::string& getName() const {
        return name_;
    }

    /**
     * Set user name
     */
    void setName(const std::string& name) {
        name_ = name;
    }

    /**
     * Get user email
     */
    const std::string& getEmail() const {
        return email_;
    }
};

/**
 * User repository for managing user data
 */
class UserRepository {
private:
    std::vector<std::shared_ptr<User>> users_;

public:
    /**
     * Add a new user
     */
    void addUser(std::shared_ptr<User> user) {
        users_.push_back(user);
    }

    /**
     * Get user by ID
     */
    std::shared_ptr<User> getUserById(int id) const {
        auto it = std::find_if(users_.begin(), users_.end(),
            [id](const std::shared_ptr<User>& user) {
                return user->getId() == id;
            });
        
        return (it != users_.end()) ? *it : nullptr;
    }

    /**
     * Get all users
     */
    const std::vector<std::shared_ptr<User>>& getAllUsers() const {
        return users_;
    }

    /**
     * Get user count
     */
    size_t getUserCount() const {
        return users_.size();
    }
};

} // namespace app
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create C++ file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "C++",
		Size:     info.Size(),
	}
}

// Large file generators for stress testing

func createLargeKotlinFile(tb testingTB, dir, name string) ScannedFile {
	content := `package com.example.app

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity

data class User(val id: Int, val name: String, val email: String)

class UserRepository {
    private val users = mutableListOf<User>()
`
	// Generate many methods
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf(`
    fun method%d(param: Int): String {
        return "Result from method %d with param $param"
    }
`, i, i)
	}
	content += "}\n"

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create large Kotlin file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Kotlin",
		Size:     info.Size(),
	}
}

func createLargeJavaFile(tb testingTB, dir, name string) ScannedFile {
	content := `package com.example.app;

import java.util.ArrayList;
import java.util.List;

public class User {
    private int id;
    private String name;
    private String email;

    public User(int id, String name, String email) {
        this.id = id;
        this.name = name;
        this.email = email;
    }
`
	// Generate many methods
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf(`
    public String method%d(int param) {
        return "Result from method %d with param " + param;
    }
`, i, i)
	}
	content += "}\n"

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create large Java file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Java",
		Size:     info.Size(),
	}
}

func createLargeSwiftFile(tb testingTB, dir, name string) ScannedFile {
	content := `import Foundation

struct User {
    let id: Int
    var name: String
    var email: String
}

class UserRepository {
`
	// Generate many methods
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf(`
    func method%d(param: Int) -> String {
        return "Result from method %d with param \\(param)"
    }
`, i, i)
	}
	content += "}\n"

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create large Swift file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "Swift",
		Size:     info.Size(),
	}
}

func createLargeCppFile(tb testingTB, dir, name string) ScannedFile {
	content := `#include <string>
#include <vector>

namespace app {

class User {
private:
    int id_;
    std::string name_;
    std::string email_;

public:
    User(int id, const std::string& name, const std::string& email)
        : id_(id), name_(name), email_(email) {}
`
	// Generate many methods
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf(`
    std::string method%d(int param) const {
        return "Result from method %d with param " + std::to_string(param);
    }
`, i, i)
	}
	content += "};\n\n} // namespace app\n"

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create large C++ file: %v", err)
	}

	info, _ := os.Stat(path)
	return ScannedFile{
		Path:     name,
		AbsPath:  path,
		Language: "C++",
		Size:     info.Size(),
	}
}
