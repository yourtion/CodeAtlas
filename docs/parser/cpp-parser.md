# C++ Parser

The C++ parser provides comprehensive support for C++ source files, including header-implementation pairing, templates, namespaces, and object-oriented features.

## Supported Features

### Symbol Extraction

- **Includes**: `#include` statements with system/local classification
- **Namespaces**: Namespace declarations and nested namespaces
- **Classes**: Class declarations with member functions and variables
- **Templates**: Template class and function declarations
- **Functions**: Function declarations and definitions
- **Methods**: Member function declarations and definitions
- **Operators**: Operator overload declarations
- **Virtual Methods**: Virtual and pure virtual method declarations
- **Constructors/Destructors**: Constructor and destructor declarations

### Documentation

- **Doxygen Comments**: Doxygen documentation extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Includes**: Classified as system (`<>`) or local (`""`)
- **Function/Method Calls**: Call relationships
- **Inheritance**: Single and multiple inheritance relationships
- **Template Instantiation**: Template usage relationships
- **Header-Implementation**: Automatic pairing of header and implementation files
- **Virtual Method Overrides**: Override relationships

## File Extensions

- `.hpp`, `.hh`, `.hxx`, `.h` - C++ header files
- `.cpp`, `.cc`, `.cxx` - C++ implementation files

## Header-Implementation Pairing

The C++ parser automatically pairs header files with their corresponding implementation files and creates cross-file relationships.

### Pairing Strategy

1. **File Name Matching**: `MyClass.hpp` pairs with `MyClass.cpp`
2. **Class-Method Matching**: Class declarations in headers match out-of-class method definitions
3. **Signature Comparison**: Method signatures are compared for matching
4. **Template Handling**: Template instantiations are linked to declarations

### Edge Types

- `implements_header`: Links .cpp file to header file
- `implements_declaration`: Links method definition to declaration
- `declares_in_header`: Links symbol to header file
- `calls_declaration`: Links method call to header declaration
- `instantiates_template`: Links template usage to template declaration

See [Header-Implementation Association](header-implementation-association.md) for detailed algorithm.

## Example Usage

### Basic C++ Class

**MyClass.hpp** (Header):
```cpp
#ifndef MYCLASS_HPP
#define MYCLASS_HPP

#include <string>
#include <vector>

/**
 * @class MyClass
 * @brief Example class demonstrating C++ features
 */
class MyClass {
public:
    /**
     * @brief Constructor
     * @param name Initial name value
     */
    MyClass(const std::string& name);
    
    /**
     * @brief Destructor
     */
    ~MyClass();
    
    /**
     * @brief Gets the name
     * @return The name string
     */
    std::string getName() const;
    
    /**
     * @brief Sets the name
     * @param name New name value
     */
    void setName(const std::string& name);
    
    /**
     * @brief Pure virtual method
     */
    virtual void process() = 0;

private:
    std::string name_;
    int id_;
};

#endif // MYCLASS_HPP
```

**MyClass.cpp** (Implementation):
```cpp
#include "MyClass.hpp"
#include <iostream>

MyClass::MyClass(const std::string& name) 
    : name_(name), id_(0) {
}

MyClass::~MyClass() {
    std::cout << "Destroying MyClass" << std::endl;
}

std::string MyClass::getName() const {
    return name_;
}

void MyClass::setName(const std::string& name) {
    name_ = name;
}
```

**Extracted Symbols**:

From **MyClass.hpp**:
- Include: `string` (system)
- Include: `vector` (system)
- Class: `MyClass`
- Constructor Declaration: `MyClass(const std::string&)`
- Destructor Declaration: `~MyClass()`
- Method Declaration: `getName() const`
- Method Declaration: `setName(const std::string&)`
- Virtual Method Declaration: `process()` (pure virtual)
- Field: `name_`, `id_`

From **MyClass.cpp**:
- Include: `MyClass.hpp` (local)
- Constructor Definition: `MyClass::MyClass(const std::string&)`
- Destructor Definition: `MyClass::~MyClass()`
- Method Definition: `MyClass::getName() const`
- Method Definition: `MyClass::setName(const std::string&)`

**Relationships**:
- `MyClass.cpp` implements_header `MyClass.hpp`
- `MyClass::MyClass` (def) implements_declaration `MyClass` (decl)
- `MyClass::getName` (def) implements_declaration `getName` (decl)
- `MyClass::setName` (def) implements_declaration `setName` (decl)

### Templates

**Container.hpp**:
```cpp
#ifndef CONTAINER_HPP
#define CONTAINER_HPP

#include <vector>

/**
 * @class Container
 * @brief Generic container template
 * @tparam T Element type
 */
template<typename T>
class Container {
public:
    /**
     * @brief Adds an element
     * @param item Element to add
     */
    void add(const T& item);
    
    /**
     * @brief Gets element at index
     * @param index Element index
     * @return Element at index
     */
    T get(size_t index) const;
    
    /**
     * @brief Gets container size
     * @return Number of elements
     */
    size_t size() const;

private:
    std::vector<T> items_;
};

// Template implementation in header
template<typename T>
void Container<T>::add(const T& item) {
    items_.push_back(item);
}

template<typename T>
T Container<T>::get(size_t index) const {
    return items_[index];
}

template<typename T>
size_t Container<T>::size() const {
    return items_.size();
}

#endif // CONTAINER_HPP
```

**Usage**:
```cpp
#include "Container.hpp"

int main() {
    Container<int> intContainer;
    intContainer.add(42);
    
    Container<std::string> stringContainer;
    stringContainer.add("hello");
    
    return 0;
}
```

**Extracted Symbols**:
- Template Class: `Container<T>`
- Template Method: `add(const T&)`
- Template Method: `get(size_t) const`
- Template Method: `size() const`

**Template Instantiations**:
- `Container<int>` instantiates_template `Container<T>`
- `Container<std::string>` instantiates_template `Container<T>`

### Namespaces

```cpp
/**
 * @namespace utils
 * @brief Utility functions namespace
 */
namespace utils {

/**
 * @namespace math
 * @brief Mathematical utilities
 */
namespace math {

/**
 * @brief Calculates square
 * @param x Input value
 * @return Square of x
 */
int square(int x);

/**
 * @brief Calculates cube
 * @param x Input value
 * @return Cube of x
 */
int cube(int x);

} // namespace math

/**
 * @namespace string
 * @brief String utilities
 */
namespace string {

/**
 * @brief Converts to uppercase
 * @param str Input string
 * @return Uppercase string
 */
std::string toUpper(const std::string& str);

} // namespace string

} // namespace utils
```

**Extracted Symbols**:
- Namespace: `utils`
- Namespace: `utils::math`
- Function: `utils::math::square`
- Function: `utils::math::cube`
- Namespace: `utils::string`
- Function: `utils::string::toUpper`

### Inheritance

```cpp
/**
 * @class Shape
 * @brief Base shape class
 */
class Shape {
public:
    virtual double area() const = 0;
    virtual double perimeter() const = 0;
    virtual ~Shape() = default;
};

/**
 * @class Rectangle
 * @brief Rectangle shape
 */
class Rectangle : public Shape {
public:
    Rectangle(double width, double height);
    
    double area() const override;
    double perimeter() const override;

private:
    double width_;
    double height_;
};

/**
 * @class Square
 * @brief Square shape (special rectangle)
 */
class Square : public Rectangle {
public:
    Square(double side);
};
```

**Extracted Relationships**:
- `Rectangle` extends `Shape`
- `Rectangle::area` overrides `Shape::area`
- `Rectangle::perimeter` overrides `Shape::perimeter`
- `Square` extends `Rectangle`

### Multiple Inheritance

```cpp
/**
 * @class Printable
 * @brief Interface for printable objects
 */
class Printable {
public:
    virtual void print() const = 0;
    virtual ~Printable() = default;
};

/**
 * @class Serializable
 * @brief Interface for serializable objects
 */
class Serializable {
public:
    virtual std::string serialize() const = 0;
    virtual ~Serializable() = default;
};

/**
 * @class Document
 * @brief Document class with multiple interfaces
 */
class Document : public Printable, public Serializable {
public:
    void print() const override;
    std::string serialize() const override;

private:
    std::string content_;
};
```

**Extracted Relationships**:
- `Document` extends `Printable`
- `Document` extends `Serializable`
- `Document::print` overrides `Printable::print`
- `Document::serialize` overrides `Serializable::serialize`

### Operator Overloading

```cpp
/**
 * @class Vector2D
 * @brief 2D vector class
 */
class Vector2D {
public:
    Vector2D(double x, double y);
    
    /**
     * @brief Addition operator
     */
    Vector2D operator+(const Vector2D& other) const;
    
    /**
     * @brief Subtraction operator
     */
    Vector2D operator-(const Vector2D& other) const;
    
    /**
     * @brief Multiplication operator
     */
    Vector2D operator*(double scalar) const;
    
    /**
     * @brief Equality operator
     */
    bool operator==(const Vector2D& other) const;

private:
    double x_;
    double y_;
};
```

**Extracted Symbols**:
- Class: `Vector2D`
- Operator: `operator+`
- Operator: `operator-`
- Operator: `operator*`
- Operator: `operator==`

## Include Classification

The C++ parser classifies includes as system or local:

**System Includes** (standard library - internal):
```cpp
#include <iostream>
#include <vector>
#include <string>
#include <algorithm>
```

**Local Includes** (project headers):
```cpp
#include "MyClass.hpp"
#include "utils/helper.hpp"
```

## Special Considerations

### Inline Methods

Inline methods in headers are marked as inline:

```cpp
class MyClass {
public:
    // Inline method
    int getValue() const { return value_; }
    
private:
    int value_;
};
```

### Out-of-Class Definitions

Out-of-class method definitions are linked to class declarations:

```cpp
// Header
class MyClass {
    void method();
};

// Implementation
void MyClass::method() {
    // Implementation
}
```

### Template Specialization

Template specializations are tracked:

```cpp
template<typename T>
class Container { };

// Specialization
template<>
class Container<bool> {
    // Specialized implementation
};
```

### Using Declarations

Using declarations are tracked:

```cpp
using std::string;
using std::vector;
using namespace std;
```

## Performance

- **Average parsing speed**: ~55 files/second
- **Memory usage**: ~10MB per 1000 lines of code
- **Header pairing overhead**: ~12% additional time
- **Template parsing**: Additional overhead for complex templates
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Complex Templates**: Highly nested template metaprogramming may not be fully analyzed
2. **SFINAE**: Substitution Failure Is Not An Error patterns have limited support
3. **Concepts**: C++20 concepts require manual handling
4. **Modules**: C++20 modules are not yet supported

## Testing

The C++ parser includes comprehensive tests:

```bash
# Run C++ parser tests
go test ./internal/parser -run TestCppParser

# Run header-implementation pairing tests
go test ./internal/parser -run TestCppHeaderImpl

# Run template tests
go test ./internal/parser -run TestCppTemplates

# Run with coverage
go test ./internal/parser -run TestCppParser -cover
```

## Example Output

For the following C++ files:

**Math.hpp**:
```cpp
#ifndef MATH_HPP
#define MATH_HPP

namespace math {
    int add(int a, int b);
    int multiply(int a, int b);
}

#endif
```

**Math.cpp**:
```cpp
#include "Math.hpp"

namespace math {
    int add(int a, int b) {
        return a + b;
    }
    
    int multiply(int a, int b) {
        return a * b;
    }
}
```

**Parsed Output**:

```json
{
  "files": [
    {
      "path": "Math.hpp",
      "language": "cpp",
      "symbols": [
        {
          "name": "math",
          "kind": "namespace",
          "line": 4
        },
        {
          "name": "math::add",
          "kind": "function_declaration",
          "line": 5,
          "signature": "int add(int a, int b)"
        },
        {
          "name": "math::multiply",
          "kind": "function_declaration",
          "line": 6,
          "signature": "int multiply(int a, int b)"
        }
      ]
    },
    {
      "path": "Math.cpp",
      "language": "cpp",
      "symbols": [
        {
          "name": "math",
          "kind": "namespace",
          "line": 3
        },
        {
          "name": "math::add",
          "kind": "function_definition",
          "line": 4,
          "signature": "int add(int a, int b)"
        },
        {
          "name": "math::multiply",
          "kind": "function_definition",
          "line": 8,
          "signature": "int multiply(int a, int b)"
        }
      ],
      "dependencies": [
        {
          "type": "implements_header",
          "source": "Math.cpp",
          "target": "Math.hpp"
        },
        {
          "type": "implements_declaration",
          "source": "math::add (def)",
          "target": "math::add (decl)"
        },
        {
          "type": "implements_declaration",
          "source": "math::multiply (def)",
          "target": "math::multiply (decl)"
        }
      ]
    }
  ]
}
```

## Related Documentation

- [C Parser](c-parser.md) - For C code
- [Header-Implementation Association](header-implementation-association.md) - Detailed pairing algorithm
- [Parser Overview](README.md) - General parser documentation
