# C Parser

The C parser provides comprehensive support for C source files, including header-implementation pairing and preprocessor directive handling.

## Supported Features

### Symbol Extraction

- **Includes**: `#include` statements with system/local classification
- **Functions**: Function declarations and definitions
- **Structs**: Struct declarations with fields
- **Unions**: Union declarations with fields
- **Enums**: Enum declarations with constants
- **Typedefs**: Type alias declarations
- **Macros**: `#define` macro definitions
- **Global Variables**: Global variable declarations

### Documentation

- **Doxygen Comments**: Doxygen documentation extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Includes**: Classified as system (`<>`) or local (`""`)
- **Function Calls**: Call relationships between functions
- **Header-Implementation**: Automatic pairing of .h and .c files
- **Type Usage**: Struct/union/enum usage relationships

## File Extensions

- `.h` - C header files (when no C++/Objective-C indicators present)
- `.c` - C implementation files

## Header-Implementation Pairing

The C parser automatically pairs header files with their corresponding implementation files and creates cross-file relationships.

### Pairing Strategy

1. **File Name Matching**: `mylib.h` pairs with `mylib.c`
2. **Declaration-Definition Matching**: Function declarations in .h match definitions in .c
3. **Signature Comparison**: Function signatures are compared for matching

### Edge Types

- `implements_header`: Links .c file to .h file
- `implements_declaration`: Links function definition to declaration
- `declares_in_header`: Links symbol to header file
- `calls_declaration`: Links function call to header declaration

See [Header-Implementation Association](header-implementation-association.md) for detailed algorithm.

## Example Usage

### Basic C Library

**math_utils.h** (Header):
```c
#ifndef MATH_UTILS_H
#define MATH_UTILS_H

#include <stddef.h>

/**
 * @brief Adds two integers
 * 
 * @param a First integer
 * @param b Second integer
 * @return Sum of a and b
 */
int add(int a, int b);

/**
 * @brief Multiplies two integers
 * 
 * @param a First integer
 * @param b Second integer
 * @return Product of a and b
 */
int multiply(int a, int b);

/**
 * @brief Calculates factorial
 * 
 * @param n Input number
 * @return Factorial of n
 */
long factorial(int n);

#endif /* MATH_UTILS_H */
```

**math_utils.c** (Implementation):
```c
#include "math_utils.h"

int add(int a, int b) {
    return a + b;
}

int multiply(int a, int b) {
    return a * b;
}

long factorial(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}
```

**Extracted Symbols**:

From **math_utils.h**:
- Include: `stddef.h` (system)
- Function Declaration: `add(int, int)`
- Function Declaration: `multiply(int, int)`
- Function Declaration: `factorial(int)`

From **math_utils.c**:
- Include: `math_utils.h` (local)
- Function Definition: `add(int, int)`
- Function Definition: `multiply(int, int)`
- Function Definition: `factorial(int)`

**Relationships**:
- `math_utils.c` implements_header `math_utils.h`
- `add` (def) implements_declaration `add` (decl)
- `multiply` (def) implements_declaration `multiply` (decl)
- `factorial` (def) implements_declaration `factorial` (decl)
- `factorial` (def) calls_declaration `factorial` (decl) (recursive call)

### Structs and Typedefs

**user.h**:
```c
#ifndef USER_H
#define USER_H

#include <stddef.h>

/**
 * @struct User
 * @brief User structure
 */
typedef struct {
    int id;
    char name[100];
    char email[100];
} User;

/**
 * @brief Creates a new user
 * 
 * @param id User ID
 * @param name User name
 * @param email User email
 * @return Pointer to new User
 */
User* create_user(int id, const char* name, const char* email);

/**
 * @brief Frees user memory
 * 
 * @param user Pointer to user
 */
void free_user(User* user);

#endif /* USER_H */
```

**user.c**:
```c
#include "user.h"
#include <stdlib.h>
#include <string.h>

User* create_user(int id, const char* name, const char* email) {
    User* user = (User*)malloc(sizeof(User));
    if (user != NULL) {
        user->id = id;
        strncpy(user->name, name, sizeof(user->name) - 1);
        strncpy(user->email, email, sizeof(user->email) - 1);
    }
    return user;
}

void free_user(User* user) {
    if (user != NULL) {
        free(user);
    }
}
```

**Extracted Symbols**:
- Typedef: `User` (struct)
- Function: `create_user`
- Function: `free_user`

### Enums

```c
/**
 * @enum Status
 * @brief Operation status codes
 */
typedef enum {
    STATUS_SUCCESS = 0,
    STATUS_ERROR = 1,
    STATUS_PENDING = 2,
    STATUS_CANCELLED = 3
} Status;
```

**Extracted Symbols**:
- Enum: `Status`
- Enum Constant: `STATUS_SUCCESS`, `STATUS_ERROR`, `STATUS_PENDING`, `STATUS_CANCELLED`

### Macros

```c
/**
 * @def MAX
 * @brief Returns maximum of two values
 */
#define MAX(a, b) ((a) > (b) ? (a) : (b))

/**
 * @def MIN
 * @brief Returns minimum of two values
 */
#define MIN(a, b) ((a) < (b) ? (a) : (b))

/**
 * @def ARRAY_SIZE
 * @brief Calculates array size
 */
#define ARRAY_SIZE(arr) (sizeof(arr) / sizeof((arr)[0]))
```

**Extracted Symbols**:
- Macro: `MAX(a, b)`
- Macro: `MIN(a, b)`
- Macro: `ARRAY_SIZE(arr)`

### Function Pointers

```c
/**
 * @typedef CompareFunc
 * @brief Function pointer for comparison
 */
typedef int (*CompareFunc)(const void*, const void*);

/**
 * @brief Sorts array using comparison function
 * 
 * @param arr Array to sort
 * @param size Array size
 * @param compare Comparison function
 */
void sort_array(void* arr, size_t size, CompareFunc compare);
```

**Extracted Symbols**:
- Typedef: `CompareFunc` (function pointer)
- Function: `sort_array`

## Include Classification

The C parser classifies includes as system or local:

**System Includes** (external):
```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
```

**Local Includes** (internal):
```c
#include "myheader.h"
#include "utils/helper.h"
```

## Special Considerations

### Static Functions

Static functions are marked as file-scope:

```c
static int helper_function(int x) {
    return x * 2;
}
```

### Extern Declarations

Extern declarations are tracked:

```c
extern int global_counter;
extern void external_function(void);
```

### Preprocessor Conditionals

Conditional compilation is tracked:

```c
#ifdef DEBUG
void debug_print(const char* msg);
#endif
```

### Header Guards

Header guards are recognized but not extracted as symbols:

```c
#ifndef MY_HEADER_H
#define MY_HEADER_H
// ...
#endif
```

## Performance

- **Average parsing speed**: ~85 files/second
- **Memory usage**: ~5MB per 1000 lines of code
- **Header pairing overhead**: ~8% additional time
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Complex Macros**: Macros with complex logic may not be fully analyzed
2. **Variadic Functions**: Variable argument functions have limited support
3. **Inline Assembly**: Assembly code blocks are not parsed

## Testing

The C parser includes comprehensive tests:

```bash
# Run C parser tests
go test ./internal/parser -run TestCParser

# Run header-implementation pairing tests
go test ./internal/parser -run TestCHeaderImpl

# Run with coverage
go test ./internal/parser -run TestCParser -cover
```

## Example Output

For the following C files:

**calculator.h**:
```c
#ifndef CALCULATOR_H
#define CALCULATOR_H

int add(int a, int b);
int subtract(int a, int b);

#endif
```

**calculator.c**:
```c
#include "calculator.h"

int add(int a, int b) {
    return a + b;
}

int subtract(int a, int b) {
    return a - b;
}
```

**Parsed Output**:

```json
{
  "files": [
    {
      "path": "calculator.h",
      "language": "c",
      "symbols": [
        {
          "name": "add",
          "kind": "function_declaration",
          "line": 4,
          "signature": "int add(int a, int b)"
        },
        {
          "name": "subtract",
          "kind": "function_declaration",
          "line": 5,
          "signature": "int subtract(int a, int b)"
        }
      ]
    },
    {
      "path": "calculator.c",
      "language": "c",
      "symbols": [
        {
          "name": "add",
          "kind": "function_definition",
          "line": 3,
          "signature": "int add(int a, int b)"
        },
        {
          "name": "subtract",
          "kind": "function_definition",
          "line": 7,
          "signature": "int subtract(int a, int b)"
        }
      ],
      "dependencies": [
        {
          "type": "implements_header",
          "source": "calculator.c",
          "target": "calculator.h"
        },
        {
          "type": "implements_declaration",
          "source": "add (def)",
          "target": "add (decl)"
        },
        {
          "type": "implements_declaration",
          "source": "subtract (def)",
          "target": "subtract (decl)"
        }
      ]
    }
  ]
}
```

## Related Documentation

- [C++ Parser](cpp-parser.md) - For C++ code
- [Objective-C Parser](objc-parser.md) - For Objective-C interop
- [Header-Implementation Association](header-implementation-association.md) - Detailed pairing algorithm
- [Parser Overview](README.md) - General parser documentation
