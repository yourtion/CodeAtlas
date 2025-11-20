# Java Parser

The Java parser provides comprehensive support for Java source files, including enterprise Java features and Android development.

## Supported Features

### Symbol Extraction

- **Packages**: Package declarations
- **Imports**: Import statements with internal/external classification
- **Classes**: Class declarations with fields, methods, constructors
- **Interfaces**: Interface declarations with method signatures
- **Enums**: Enum declarations with constants
- **Annotations**: Annotation definitions and usage
- **Inner Classes**: Nested and inner class declarations
- **Anonymous Classes**: Anonymous class instantiations

### Documentation

- **Javadoc**: Complete Javadoc comment extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Imports**: Classified as internal (java.*, javax.*) or external
- **Method Calls**: Call relationships between methods
- **Inheritance**: Class extension and interface implementation
- **Annotation Usage**: Annotation application relationships
- **Generics**: Type parameter relationships

## File Extensions

- `.java` - Java source files

## Example Usage

### Basic Java Class

```java
package com.example.app;

import java.util.List;
import java.util.ArrayList;

/**
 * User entity representing a user in the system
 * 
 * @author CodeAtlas
 * @version 1.0
 */
public class User {
    private Long id;
    private String name;
    private String email;
    
    /**
     * Creates a new User instance
     * 
     * @param id the user ID
     * @param name the user name
     * @param email the user email
     */
    public User(Long id, String name, String email) {
        this.id = id;
        this.name = name;
        this.email = email;
    }
    
    /**
     * Gets the user ID
     * 
     * @return the user ID
     */
    public Long getId() {
        return id;
    }
    
    // Additional getters and setters...
}
```

**Extracted Symbols**:
- Package: `com.example.app`
- Import: `java.util.List` (internal)
- Import: `java.util.ArrayList` (internal)
- Class: `User`
- Field: `id`, `name`, `email`
- Constructor: `User(Long, String, String)`
- Method: `getId()`

### Annotations

```java
package com.example.annotations;

import java.lang.annotation.*;

/**
 * Custom annotation for marking API endpoints
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
public @interface ApiEndpoint {
    String path();
    String method() default "GET";
}

// Usage
public class UserController {
    @ApiEndpoint(path = "/users", method = "GET")
    public List<User> getUsers() {
        return new ArrayList<>();
    }
}
```

**Extracted Symbols**:
- Annotation: `ApiEndpoint`
- Class: `UserController`
- Method: `getUsers` (annotated with `@ApiEndpoint`)

### Enums

```java
package com.example.enums;

/**
 * User role enumeration
 */
public enum UserRole {
    ADMIN("Administrator"),
    USER("Regular User"),
    GUEST("Guest User");
    
    private final String displayName;
    
    UserRole(String displayName) {
        this.displayName = displayName;
    }
    
    public String getDisplayName() {
        return displayName;
    }
}
```

**Extracted Symbols**:
- Enum: `UserRole`
- Enum Constant: `ADMIN`, `USER`, `GUEST`
- Field: `displayName`
- Constructor: `UserRole(String)`
- Method: `getDisplayName()`

### Generics

```java
package com.example.generics;

import java.util.List;

/**
 * Generic repository interface
 * 
 * @param <T> the entity type
 * @param <ID> the ID type
 */
public interface Repository<T, ID> {
    T findById(ID id);
    List<T> findAll();
    void save(T entity);
    void delete(ID id);
}

public class UserRepository implements Repository<User, Long> {
    @Override
    public User findById(Long id) {
        return null;
    }
    
    @Override
    public List<User> findAll() {
        return new ArrayList<>();
    }
    
    @Override
    public void save(User entity) {
        // Implementation
    }
    
    @Override
    public void delete(Long id) {
        // Implementation
    }
}
```

**Extracted Symbols**:
- Interface: `Repository<T, ID>`
- Class: `UserRepository implements Repository<User, Long>`
- Methods with generic types preserved

## Import Classification

The Java parser classifies imports as internal or external:

**Internal** (Java standard library):
- `java.*`
- `javax.*`

**External** (third-party libraries):
- `org.springframework.*`
- `com.google.android.*`
- `androidx.*`
- All other imports

## Special Considerations

### Inner Classes

Inner classes are properly extracted with their enclosing class context:

```java
public class OuterClass {
    private class InnerClass {
        void innerMethod() {}
    }
    
    static class StaticNestedClass {
        void nestedMethod() {}
    }
}
```

### Anonymous Classes

Anonymous classes are identified but may have generated names:

```java
Runnable runnable = new Runnable() {
    @Override
    public void run() {
        System.out.println("Running");
    }
};
```

### Lambda Expressions

Lambda expressions are tracked as method references:

```java
List<String> names = users.stream()
    .map(User::getName)
    .collect(Collectors.toList());
```

## Performance

- **Average parsing speed**: ~75 files/second
- **Memory usage**: ~6MB per 1000 lines of code
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Records**: Java 14+ record classes have limited support
2. **Pattern Matching**: Java 16+ pattern matching may not be fully supported
3. **Sealed Classes**: Java 17+ sealed classes require manual handling

## Testing

The Java parser includes comprehensive tests:

```bash
# Run Java parser tests
go test ./internal/parser -run TestJavaParser

# Run with coverage
go test ./internal/parser -run TestJavaParser -cover
```

## Example Output

For the following Java file:

```java
package com.example;

public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
```

**Parsed Output**:

```json
{
  "path": "Calculator.java",
  "language": "java",
  "symbols": [
    {
      "name": "com.example",
      "kind": "package",
      "line": 1
    },
    {
      "name": "Calculator",
      "kind": "class",
      "line": 3,
      "signature": "public class Calculator"
    },
    {
      "name": "add",
      "kind": "method",
      "line": 4,
      "signature": "public int add(int a, int b)"
    }
  ],
  "dependencies": []
}
```

## Related Documentation

- [Kotlin Parser](kotlin-parser.md) - For Kotlin interop
- [Parser Overview](README.md) - General parser documentation
- [Android Development](../examples/android-example.md) - Android-specific examples
