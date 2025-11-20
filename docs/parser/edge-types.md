# Edge Types Reference

This document describes all edge (relationship) types used in the CodeAtlas knowledge graph.

## Overview

Edges represent relationships between code entities (symbols, files, modules). They form the foundation of the knowledge graph, enabling dependency analysis, call graph traversal, and code navigation.

## Edge Type Categories

### 1. Import/Include Relationships

Edges representing file-level dependencies through import or include statements.

#### `import`

**Description**: A file imports a module, package, or another file.

**Source**: File or symbol
**Target**: Module, package, or file
**Languages**: Go, JavaScript, TypeScript, Python, Kotlin, Java, Swift

**Examples**:

```javascript
// JavaScript
import { User } from './models/User';
// Edge: current_file --import--> ./models/User
```

```python
# Python
import numpy as np
# Edge: current_file --import--> numpy (external)
```

```kotlin
// Kotlin
import com.example.app.User
// Edge: current_file --import--> com.example.app.User
```

**Properties**:
- `external`: Boolean indicating if import is external (third-party library)
- `source_file`: Path to importing file
- `target_module`: Module/package name

#### `includes`

**Description**: A file includes another file (C/C++/Objective-C).

**Source**: File
**Target**: File
**Languages**: C, C++, Objective-C

**Examples**:

```c
// C
#include <stdio.h>
// Edge: current_file --includes--> stdio.h (system, external)

#include "myheader.h"
// Edge: current_file --includes--> myheader.h (local, internal)
```

```cpp
// C++
#include <vector>
// Edge: current_file --includes--> vector (system, internal)

#include "MyClass.hpp"
// Edge: current_file --includes--> MyClass.hpp (local, internal)
```

**Properties**:
- `external`: Boolean indicating if include is system header
- `system`: Boolean indicating if using angle brackets (`<>`)

### 2. Call Relationships

Edges representing function/method invocations.

#### `call`

**Description**: A function/method calls another function/method.

**Source**: Function/method symbol
**Target**: Function/method symbol
**Languages**: All supported languages

**Examples**:

```go
// Go
func main() {
    result := add(1, 2)
}
// Edge: main --call--> add
```

```swift
// Swift
func processUser() {
    let user = fetchUser()
}
// Edge: processUser --call--> fetchUser
```

**Properties**:
- `source_file`: File containing the caller
- `target_file`: File containing the callee
- `line`: Line number of the call

#### `calls_declaration`

**Description**: A function/method calls a function/method declared in a header file (C/C++/Objective-C).

**Source**: Function/method symbol
**Target**: Function/method declaration in header
**Languages**: C, C++, Objective-C

**Examples**:

```c
// main.c
#include "math.h"

int main() {
    int result = add(1, 2);  // Calls declaration in math.h
}
// Edge: main --calls_declaration--> add (in math.h)
```

**Properties**:
- `source_file`: Implementation file containing the call
- `target_file`: Header file containing the declaration
- `line`: Line number of the call

**Why this matters**: Links calls to header declarations rather than implementations, providing a more accurate dependency graph.

### 3. Inheritance Relationships

Edges representing class/interface inheritance and implementation.

#### `extends`

**Description**: A class extends (inherits from) another class.

**Source**: Derived class
**Target**: Base class
**Languages**: Go, JavaScript, TypeScript, Python, Kotlin, Java, Swift, Objective-C, C++

**Examples**:

```java
// Java
public class Dog extends Animal {
}
// Edge: Dog --extends--> Animal
```

```cpp
// C++
class Rectangle : public Shape {
};
// Edge: Rectangle --extends--> Shape
```

```swift
// Swift
class ViewController: UIViewController {
}
// Edge: ViewController --extends--> UIViewController
```

**Properties**:
- `access`: Access modifier (public, private, protected) for C++
- `virtual`: Boolean indicating virtual inheritance (C++)

#### `implements`

**Description**: A class implements an interface.

**Source**: Class
**Target**: Interface
**Languages**: Java, Kotlin, TypeScript

**Examples**:

```java
// Java
public class UserRepository implements Repository<User> {
}
// Edge: UserRepository --implements--> Repository
```

```kotlin
// Kotlin
class UserService : UserInterface {
}
// Edge: UserService --implements--> UserInterface
```

**Properties**:
- `generic_types`: Type parameters for generic interfaces

#### `conforms`

**Description**: A class/struct conforms to a protocol (Swift/Objective-C).

**Source**: Class/struct
**Target**: Protocol
**Languages**: Swift, Objective-C

**Examples**:

```swift
// Swift
struct User: Codable {
}
// Edge: User --conforms--> Codable
```

```objc
// Objective-C
@interface User : NSObject <NSCoding>
@end
// Edge: User --conforms--> NSCoding
```

### 4. Header-Implementation Relationships

Edges specific to languages with header/implementation file separation.

#### `implements_header`

**Description**: An implementation file implements declarations from a header file.

**Source**: Implementation file (.c, .cpp, .m)
**Target**: Header file (.h, .hpp)
**Languages**: C, C++, Objective-C

**Examples**:

```
MyClass.cpp --implements_header--> MyClass.hpp
math.c --implements_header--> math.h
User.m --implements_header--> User.h
```

**Properties**:
- `source_file`: Implementation file path
- `target_file`: Header file path

**Use Cases**:
- Navigate from implementation to header
- Find all implementations of a header
- Verify header-implementation consistency

#### `implements_declaration`

**Description**: A symbol implementation implements a symbol declaration.

**Source**: Symbol in implementation file
**Target**: Symbol in header file
**Languages**: C, C++, Objective-C

**Examples**:

```c
// math.h
int add(int a, int b);  // Declaration

// math.c
int add(int a, int b) {  // Implementation
    return a + b;
}
// Edge: add (impl) --implements_declaration--> add (decl)
```

```cpp
// MyClass.hpp
class MyClass {
    void method();  // Declaration
};

// MyClass.cpp
void MyClass::method() {  // Implementation
}
// Edge: MyClass::method (impl) --implements_declaration--> MyClass::method (decl)
```

**Properties**:
- `source_symbol`: Implementation symbol ID
- `target_symbol`: Declaration symbol ID
- `source_file`: Implementation file
- `target_file`: Header file

#### `declares_in_header`

**Description**: A symbol is declared in a header file.

**Source**: Symbol
**Target**: Header file
**Languages**: C, C++, Objective-C

**Examples**:

```
add (function) --declares_in_header--> math.h
MyClass (class) --declares_in_header--> MyClass.hpp
```

**Properties**:
- `symbol_id`: Symbol ID
- `file_id`: Header file ID

**Use Cases**:
- Find where a symbol is declared
- List all symbols declared in a header
- Navigate from symbol to its declaration

### 5. Template/Generic Relationships

Edges representing template instantiation and generic type usage.

#### `instantiates_template`

**Description**: A template instantiation uses a template declaration.

**Source**: Template instantiation
**Target**: Template declaration
**Languages**: C++

**Examples**:

```cpp
// Template declaration
template<typename T>
class Container { };

// Template instantiation
Container<int> intContainer;
// Edge: Container<int> --instantiates_template--> Container<T>

Container<std::string> stringContainer;
// Edge: Container<std::string> --instantiates_template--> Container<T>
```

**Properties**:
- `type_arguments`: List of type arguments used in instantiation
- `source_file`: File containing instantiation
- `target_file`: File containing template declaration

### 6. Extension Relationships

Edges representing type extensions.

#### `extends_type`

**Description**: An extension extends a type (Swift/Kotlin).

**Source**: Extension
**Target**: Type being extended
**Languages**: Swift, Kotlin

**Examples**:

```swift
// Swift
extension String {
    func isValidEmail() -> Bool {
        return self.contains("@")
    }
}
// Edge: String (extension) --extends_type--> String (type)
```

```kotlin
// Kotlin
fun String.isValidEmail(): Boolean {
    return this.contains("@")
}
// Edge: String.isValidEmail --extends_type--> String
```

**Properties**:
- `extension_file`: File containing the extension
- `type_file`: File containing the original type (if available)

### 7. Annotation/Decorator Relationships

Edges representing annotation or decorator usage.

#### `annotated_with`

**Description**: A symbol is annotated with an annotation/decorator.

**Source**: Symbol (class, method, field)
**Target**: Annotation/decorator
**Languages**: Java, Kotlin, Python, TypeScript

**Examples**:

```java
// Java
@Override
public void method() {
}
// Edge: method --annotated_with--> Override
```

```python
# Python
@property
def name(self):
    return self._name
// Edge: name --annotated_with--> property
```

**Properties**:
- `annotation_arguments`: Arguments passed to annotation

### 8. Reference Relationships

Edges representing general symbol references.

#### `reference`

**Description**: A symbol references another symbol (variable usage, type reference, etc.).

**Source**: Symbol
**Target**: Symbol
**Languages**: All supported languages

**Examples**:

```go
// Go
type User struct {
    ID int
}

func getUser() *User {  // References User type
    return &User{}
}
// Edge: getUser --reference--> User
```

**Properties**:
- `reference_type`: Type of reference (type, variable, constant)
- `line`: Line number of reference

### 9. Override Relationships

Edges representing method overrides.

#### `overrides`

**Description**: A method overrides a method from a base class.

**Source**: Overriding method
**Target**: Overridden method
**Languages**: Java, Kotlin, Swift, C++

**Examples**:

```java
// Java
class Animal {
    void makeSound() { }
}

class Dog extends Animal {
    @Override
    void makeSound() { }  // Overrides Animal.makeSound
}
// Edge: Dog.makeSound --overrides--> Animal.makeSound
```

```cpp
// C++
class Shape {
    virtual double area() = 0;
};

class Rectangle : public Shape {
    double area() override {  // Overrides Shape.area
        return width * height;
    }
};
// Edge: Rectangle::area --overrides--> Shape::area
```

## Edge Properties

All edges can have the following common properties:

| Property | Type | Description |
|----------|------|-------------|
| `edge_id` | UUID | Unique identifier for the edge |
| `source_id` | UUID | Source entity ID |
| `target_id` | UUID | Target entity ID |
| `edge_type` | String | Type of relationship (see above) |
| `repository_id` | Integer | Repository containing this edge |
| `created_at` | Timestamp | When edge was created |
| `updated_at` | Timestamp | When edge was last updated |

Additional properties are edge-type specific (see descriptions above).

## Querying Edges

### SQL Examples

**Find all calls from a function**:
```sql
SELECT e.*, s.name as target_name
FROM edges e
JOIN symbols s ON e.target_id = s.id
WHERE e.source_id = <function_id>
  AND e.edge_type = 'call';
```

**Find implementation for a header**:
```sql
SELECT f.*
FROM files f
JOIN edges e ON e.source_id = f.id
WHERE e.target_id = <header_file_id>
  AND e.edge_type = 'implements_header';
```

**Find all classes implementing an interface**:
```sql
SELECT s.*
FROM symbols s
JOIN edges e ON e.source_id = s.id
WHERE e.target_id = <interface_id>
  AND e.edge_type = 'implements';
```

**Find all overrides of a method**:
```sql
SELECT s.*
FROM symbols s
JOIN edges e ON e.source_id = s.id
WHERE e.target_id = <method_id>
  AND e.edge_type = 'overrides';
```

### API Examples

**Get callers of a function**:
```bash
curl http://localhost:8080/api/v1/symbols/<symbol_id>/callers
```

**Get callees of a function**:
```bash
curl http://localhost:8080/api/v1/symbols/<symbol_id>/callees
```

**Get dependencies of a file**:
```bash
curl http://localhost:8080/api/v1/files/<file_id>/dependencies
```

## Edge Type Summary

| Edge Type | Source | Target | Languages | Description |
|-----------|--------|--------|-----------|-------------|
| `import` | File/Symbol | Module/File | Go, JS, TS, Python, Kotlin, Java, Swift | Import statement |
| `includes` | File | File | C, C++, Objective-C | Include directive |
| `call` | Function | Function | All | Function call |
| `calls_declaration` | Function | Function (header) | C, C++, Objective-C | Call to header declaration |
| `extends` | Class | Class | Most OOP languages | Class inheritance |
| `implements` | Class | Interface | Java, Kotlin, TypeScript | Interface implementation |
| `conforms` | Class/Struct | Protocol | Swift, Objective-C | Protocol conformance |
| `implements_header` | File | File | C, C++, Objective-C | Implementation-header pairing |
| `implements_declaration` | Symbol | Symbol | C, C++, Objective-C | Implementation-declaration pairing |
| `declares_in_header` | Symbol | File | C, C++, Objective-C | Symbol declared in header |
| `instantiates_template` | Instantiation | Template | C++ | Template instantiation |
| `extends_type` | Extension | Type | Swift, Kotlin | Type extension |
| `annotated_with` | Symbol | Annotation | Java, Kotlin, Python, TS | Annotation usage |
| `reference` | Symbol | Symbol | All | General reference |
| `overrides` | Method | Method | Java, Kotlin, Swift, C++ | Method override |

## Best Practices

### 1. Use Specific Edge Types

Prefer specific edge types over generic ones:

✅ Good:
```
UserRepository --implements--> Repository
```

❌ Avoid:
```
UserRepository --reference--> Repository
```

### 2. Maintain Bidirectional Relationships

For navigation, consider creating reverse edges:

```
// Forward edge
Dog --extends--> Animal

// Reverse edge (for efficient queries)
Animal --extended_by--> Dog
```

### 3. Include Context Properties

Add properties to edges for better context:

```json
{
  "edge_type": "call",
  "source_id": "func_123",
  "target_id": "func_456",
  "line": 42,
  "source_file": "main.go",
  "target_file": "utils.go"
}
```

### 4. Handle External Dependencies

Mark external dependencies appropriately:

```json
{
  "edge_type": "import",
  "source_id": "file_123",
  "target_module": "numpy",
  "external": true
}
```

## Related Documentation

- [Parser Overview](README.md)
- [Header-Implementation Association](header-implementation-association.md)
- [Database Schema](../schema.md)
- [API Reference](../api/README.md)
