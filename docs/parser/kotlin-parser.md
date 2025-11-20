# Kotlin Parser

The Kotlin parser provides comprehensive support for Kotlin source files, including Android-specific features and Kotlin-specific constructs.

## Supported Features

### Symbol Extraction

- **Packages**: Package declarations
- **Imports**: Import statements with internal/external classification
- **Classes**: Regular classes, data classes, sealed classes, object declarations
- **Interfaces**: Interface declarations with method signatures
- **Functions**: Top-level functions, member functions, extension functions
- **Properties**: Top-level properties, member properties, extension properties
- **Annotations**: Annotation usage and definitions
- **Companion Objects**: Companion object declarations

### Documentation

- **KDoc**: Kotlin documentation comments extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Imports**: Classified as internal (kotlin.*, kotlinx.*) or external
- **Function Calls**: Call relationships between functions
- **Inheritance**: Class extension and interface implementation
- **Extension Relationships**: Extension function/property targets

## File Extensions

- `.kt` - Kotlin source files
- `.kts` - Kotlin script files

## Example Usage

### Basic Kotlin Class

```kotlin
package com.example.app

import kotlinx.coroutines.*

/**
 * User data class representing a user entity
 */
data class User(
    val id: Long,
    val name: String,
    val email: String
)

/**
 * User repository for data access
 */
class UserRepository {
    suspend fun getUser(id: Long): User? {
        return withContext(Dispatchers.IO) {
            // Database access
            null
        }
    }
}
```

**Extracted Symbols**:
- Package: `com.example.app`
- Import: `kotlinx.coroutines.*` (internal)
- Class: `User` (data class)
- Class: `UserRepository`
- Function: `getUser` (suspend function)

### Extension Functions

```kotlin
package com.example.extensions

fun String.isValidEmail(): Boolean {
    return this.contains("@")
}

fun List<Int>.sum(): Int {
    return this.fold(0) { acc, i -> acc + i }
}
```

**Extracted Symbols**:
- Extension Function: `String.isValidEmail`
- Extension Function: `List<Int>.sum`

### Sealed Classes

```kotlin
package com.example.state

sealed class Result<out T> {
    data class Success<T>(val data: T) : Result<T>()
    data class Error(val message: String) : Result<Nothing>()
    object Loading : Result<Nothing>()
}
```

**Extracted Symbols**:
- Sealed Class: `Result`
- Data Class: `Success` (nested)
- Data Class: `Error` (nested)
- Object: `Loading` (nested)

## Import Classification

The Kotlin parser classifies imports as internal or external:

**Internal** (Kotlin standard library):
- `kotlin.*`
- `kotlinx.*`

**External** (third-party libraries):
- `com.google.android.*`
- `androidx.*`
- `com.squareup.*`
- All other imports

## Special Considerations

### Coroutines

Suspend functions are properly identified:

```kotlin
suspend fun fetchData(): String {
    delay(1000)
    return "data"
}
```

### Nullable Types

Nullable types are preserved in signatures:

```kotlin
fun findUser(id: Long): User? {
    return null
}
```

### Companion Objects

Companion objects are extracted as separate symbols:

```kotlin
class MyClass {
    companion object {
        const val TAG = "MyClass"
        fun create(): MyClass = MyClass()
    }
}
```

## Performance

- **Average parsing speed**: ~70 files/second
- **Memory usage**: ~5MB per 1000 lines of code
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Inline classes**: Limited support for inline value classes
2. **Context receivers**: Experimental features may not be fully supported
3. **Multiplatform**: Expect declarations require manual handling

## Testing

The Kotlin parser includes comprehensive tests:

```bash
# Run Kotlin parser tests
go test ./internal/parser -run TestKotlinParser

# Run with coverage
go test ./internal/parser -run TestKotlinParser -cover
```

## Example Output

For the following Kotlin file:

```kotlin
package com.example

data class Person(val name: String, val age: Int)

fun greet(person: Person) {
    println("Hello, ${person.name}")
}
```

**Parsed Output**:

```json
{
  "path": "Person.kt",
  "language": "kotlin",
  "symbols": [
    {
      "name": "com.example",
      "kind": "package",
      "line": 1
    },
    {
      "name": "Person",
      "kind": "data_class",
      "line": 3,
      "signature": "data class Person(val name: String, val age: Int)"
    },
    {
      "name": "greet",
      "kind": "function",
      "line": 5,
      "signature": "fun greet(person: Person)"
    }
  ],
  "dependencies": [
    {
      "type": "call",
      "source": "greet",
      "target": "println"
    }
  ]
}
```

## Related Documentation

- [Java Parser](java-parser.md) - For Java interop
- [Parser Overview](README.md) - General parser documentation
- [Android Development](../examples/android-example.md) - Android-specific examples
