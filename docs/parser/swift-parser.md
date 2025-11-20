# Swift Parser

The Swift parser provides comprehensive support for Swift source files, including iOS, macOS, and SwiftUI development.

## Supported Features

### Symbol Extraction

- **Imports**: Import statements with framework classification
- **Classes**: Class declarations with properties and methods
- **Structs**: Struct declarations with properties and methods
- **Enums**: Enum declarations with associated values and raw values
- **Protocols**: Protocol declarations with requirements
- **Extensions**: Extension declarations with extended type tracking
- **Functions**: Global functions and member functions
- **Properties**: Stored properties, computed properties, property observers
- **Subscripts**: Subscript declarations

### Documentation

- **Swift Documentation**: Swift documentation comment extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Imports**: Classified as external (Foundation, UIKit, SwiftUI) or internal
- **Function Calls**: Call relationships including optional chaining
- **Protocol Conformance**: Protocol adoption relationships
- **Extension Relationships**: Extension-to-type associations
- **Inheritance**: Class inheritance relationships

## File Extensions

- `.swift` - Swift source files

## Example Usage

### Basic Swift Class

```swift
import Foundation

/**
 User model representing a user entity
 
 - Author: CodeAtlas
 - Version: 1.0
 */
class User {
    let id: Int
    var name: String
    var email: String
    
    /**
     Initializes a new User instance
     
     - Parameters:
       - id: The user ID
       - name: The user name
       - email: The user email
     */
    init(id: Int, name: String, email: String) {
        self.id = id
        self.name = name
        self.email = email
    }
    
    /**
     Validates the user's email address
     
     - Returns: true if email is valid, false otherwise
     */
    func validateEmail() -> Bool {
        return email.contains("@")
    }
}
```

**Extracted Symbols**:
- Import: `Foundation` (external)
- Class: `User`
- Property: `id`, `name`, `email`
- Method: `init(id:name:email:)`
- Method: `validateEmail()`

### Protocols and Extensions

```swift
import Foundation

/**
 Protocol defining identifiable entities
 */
protocol Identifiable {
    var id: Int { get }
}

/**
 Extension adding Identifiable conformance to User
 */
extension User: Identifiable {
    // id property already exists
}

/**
 Extension adding utility methods to String
 */
extension String {
    func isValidEmail() -> Bool {
        return self.contains("@")
    }
}
```

**Extracted Symbols**:
- Protocol: `Identifiable`
- Extension: `User` (conforms to `Identifiable`)
- Extension: `String`
- Method: `isValidEmail()` (extension method)

### Enums with Associated Values

```swift
/**
 Result type for async operations
 */
enum Result<T> {
    case success(T)
    case failure(Error)
    case loading
    
    /**
     Checks if the result is successful
     
     - Returns: true if success, false otherwise
     */
    func isSuccess() -> Bool {
        if case .success = self {
            return true
        }
        return false
    }
}
```

**Extracted Symbols**:
- Enum: `Result<T>`
- Case: `success(T)`, `failure(Error)`, `loading`
- Method: `isSuccess()`

### Property Observers

```swift
class ViewController {
    var title: String = "" {
        willSet {
            print("Will set title to \(newValue)")
        }
        didSet {
            print("Did set title from \(oldValue)")
            updateUI()
        }
    }
    
    func updateUI() {
        // Update UI
    }
}
```

**Extracted Symbols**:
- Class: `ViewController`
- Property: `title` (with observers)
- Method: `updateUI()`

### SwiftUI Views

```swift
import SwiftUI

/**
 Main content view for the app
 */
struct ContentView: View {
    @State private var count: Int = 0
    
    var body: some View {
        VStack {
            Text("Count: \(count)")
            Button("Increment") {
                count += 1
            }
        }
    }
}
```

**Extracted Symbols**:
- Import: `SwiftUI` (external)
- Struct: `ContentView` (conforms to `View`)
- Property: `count` (with `@State` attribute)
- Property: `body` (computed property)

## Import Classification

The Swift parser classifies imports as internal or external:

**External** (Apple frameworks):
- `Foundation`
- `UIKit`
- `SwiftUI`
- `Combine`
- `CoreData`
- All Apple framework imports

**Internal** (project modules):
- Custom module imports
- Relative imports

## Special Considerations

### Optional Chaining

Optional chaining is preserved in call relationships:

```swift
let name = user?.profile?.name
```

### Closures

Closures are tracked as inline functions:

```swift
let numbers = [1, 2, 3, 4, 5]
let doubled = numbers.map { $0 * 2 }
```

### Property Wrappers

Property wrappers like `@State`, `@Binding`, `@Published` are preserved:

```swift
@State private var isPresented: Bool = false
@Published var users: [User] = []
```

### Async/Await

Async functions are properly identified:

```swift
func fetchData() async throws -> Data {
    let (data, _) = try await URLSession.shared.data(from: url)
    return data
}
```

## Performance

- **Average parsing speed**: ~65 files/second
- **Memory usage**: ~7MB per 1000 lines of code
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Result Builders**: Complex result builder syntax may not be fully supported
2. **Macros**: Swift 5.9+ macros require manual handling
3. **Actors**: Swift 5.5+ actor types have limited support

## Testing

The Swift parser includes comprehensive tests:

```bash
# Run Swift parser tests
go test ./internal/parser -run TestSwiftParser

# Run with coverage
go test ./internal/parser -run TestSwiftParser -cover
```

## Example Output

For the following Swift file:

```swift
import Foundation

struct Person {
    let name: String
    let age: Int
}

func greet(person: Person) {
    print("Hello, \(person.name)")
}
```

**Parsed Output**:

```json
{
  "path": "Person.swift",
  "language": "swift",
  "symbols": [
    {
      "name": "Foundation",
      "kind": "import",
      "line": 1
    },
    {
      "name": "Person",
      "kind": "struct",
      "line": 3,
      "signature": "struct Person"
    },
    {
      "name": "name",
      "kind": "property",
      "line": 4
    },
    {
      "name": "age",
      "kind": "property",
      "line": 5
    },
    {
      "name": "greet",
      "kind": "function",
      "line": 8,
      "signature": "func greet(person: Person)"
    }
  ],
  "dependencies": [
    {
      "type": "import",
      "source": "Person.swift",
      "target": "Foundation",
      "external": true
    },
    {
      "type": "call",
      "source": "greet",
      "target": "print"
    }
  ]
}
```

## Related Documentation

- [Objective-C Parser](objc-parser.md) - For Objective-C interop
- [Parser Overview](README.md) - General parser documentation
- [iOS Development](../examples/ios-example.md) - iOS-specific examples
