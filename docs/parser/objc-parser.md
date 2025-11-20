# Objective-C Parser

The Objective-C parser provides comprehensive support for Objective-C source files, including header-implementation pairing and mixed Swift/Objective-C codebases.

## Supported Features

### Symbol Extraction

- **Imports**: `#import` and `#include` statements
- **Interfaces**: `@interface` declarations in header files
- **Implementations**: `@implementation` declarations in implementation files
- **Protocols**: `@protocol` declarations
- **Categories**: Category declarations and implementations
- **Properties**: `@property` declarations with attributes
- **Methods**: Instance and class method declarations/implementations
- **Blocks**: Block type declarations

### Documentation

- **Header Documentation**: Documentation comment extraction
- **Inline Comments**: Single-line and multi-line comments

### Dependencies

- **Imports**: Classified as external (Foundation, UIKit, CoreData) or internal
- **Message Sends**: Method invocation relationships
- **Protocol Conformance**: Protocol adoption relationships
- **Inheritance**: Class inheritance relationships
- **Header-Implementation**: Automatic pairing of .h and .m files

## File Extensions

- `.h` - Objective-C header files
- `.m` - Objective-C implementation files
- `.mm` - Objective-C++ implementation files

## Header-Implementation Pairing

The Objective-C parser automatically pairs header files with their corresponding implementation files and creates cross-file relationships.

### Pairing Strategy

1. **File Name Matching**: `MyClass.h` pairs with `MyClass.m`
2. **Interface-Implementation Matching**: `@interface` in .h matches `@implementation` in .m
3. **Method Matching**: Method declarations in @interface match implementations in @implementation

### Edge Types

- `implements_header`: Links .m file to .h file
- `implements_declaration`: Links method implementation to declaration
- `declares_in_header`: Links symbol to header file

See [Header-Implementation Association](header-implementation-association.md) for detailed algorithm.

## Example Usage

### Basic Objective-C Class

**MyClass.h** (Header):
```objc
#import <Foundation/Foundation.h>

/**
 * User model representing a user entity
 */
@interface User : NSObject

@property (nonatomic, strong) NSString *name;
@property (nonatomic, strong) NSString *email;
@property (nonatomic, assign) NSInteger userId;

/**
 * Initializes a new User instance
 *
 * @param name The user name
 * @param email The user email
 * @return A new User instance
 */
- (instancetype)initWithName:(NSString *)name email:(NSString *)email;

/**
 * Validates the user's email address
 *
 * @return YES if email is valid, NO otherwise
 */
- (BOOL)validateEmail;

@end
```

**MyClass.m** (Implementation):
```objc
#import "User.h"

@implementation User

- (instancetype)initWithName:(NSString *)name email:(NSString *)email {
    self = [super init];
    if (self) {
        _name = name;
        _email = email;
        _userId = 0;
    }
    return self;
}

- (BOOL)validateEmail {
    return [self.email containsString:@"@"];
}

@end
```

**Extracted Symbols**:

From **User.h**:
- Import: `Foundation/Foundation.h` (external)
- Interface: `User` (extends `NSObject`)
- Property: `name`, `email`, `userId`
- Method Declaration: `initWithName:email:`
- Method Declaration: `validateEmail`

From **User.m**:
- Import: `User.h` (internal)
- Implementation: `User`
- Method Implementation: `initWithName:email:`
- Method Implementation: `validateEmail`

**Relationships**:
- `User.m` implements_header `User.h`
- `initWithName:email:` (impl) implements_declaration `initWithName:email:` (decl)
- `validateEmail` (impl) implements_declaration `validateEmail` (decl)

### Protocols

**UserDelegate.h**:
```objc
#import <Foundation/Foundation.h>

@class User;

/**
 * Protocol for user-related delegate methods
 */
@protocol UserDelegate <NSObject>

@required
- (void)userDidLogin:(User *)user;
- (void)userDidLogout:(User *)user;

@optional
- (void)userDidUpdateProfile:(User *)user;

@end
```

**Extracted Symbols**:
- Protocol: `UserDelegate` (conforms to `NSObject`)
- Method: `userDidLogin:` (required)
- Method: `userDidLogout:` (required)
- Method: `userDidUpdateProfile:` (optional)

### Categories

**NSString+Validation.h**:
```objc
#import <Foundation/Foundation.h>

/**
 * Category adding validation methods to NSString
 */
@interface NSString (Validation)

- (BOOL)isValidEmail;
- (BOOL)isValidPhoneNumber;

@end
```

**NSString+Validation.m**:
```objc
#import "NSString+Validation.h"

@implementation NSString (Validation)

- (BOOL)isValidEmail {
    return [self containsString:@"@"];
}

- (BOOL)isValidPhoneNumber {
    NSCharacterSet *numbers = [NSCharacterSet decimalDigitCharacterSet];
    return [[self stringByTrimmingCharactersInSet:numbers] length] == 0;
}

@end
```

**Extracted Symbols**:
- Category: `NSString (Validation)`
- Method: `isValidEmail`
- Method: `isValidPhoneNumber`

### Property Attributes

```objc
@interface MyClass : NSObject

@property (nonatomic, strong) NSString *strongProperty;
@property (nonatomic, weak) id<MyDelegate> weakProperty;
@property (nonatomic, copy) NSString *copyProperty;
@property (nonatomic, assign) NSInteger assignProperty;
@property (nonatomic, readonly) NSString *readonlyProperty;

@end
```

**Extracted Properties** with attributes:
- `strongProperty` (strong)
- `weakProperty` (weak)
- `copyProperty` (copy)
- `assignProperty` (assign)
- `readonlyProperty` (readonly)

### Blocks

```objc
typedef void (^CompletionBlock)(BOOL success, NSError *error);

@interface NetworkManager : NSObject

- (void)fetchDataWithCompletion:(CompletionBlock)completion;

@end
```

**Extracted Symbols**:
- Typedef: `CompletionBlock` (block type)
- Class: `NetworkManager`
- Method: `fetchDataWithCompletion:`

## Import Classification

The Objective-C parser classifies imports as internal or external:

**External** (Apple frameworks):
- `Foundation/Foundation.h`
- `UIKit/UIKit.h`
- `CoreData/CoreData.h`
- All system framework imports

**Internal** (project headers):
- `"MyClass.h"`
- Relative path imports

## Special Considerations

### Message Sends

Message sends are tracked as call relationships:

```objc
[user validateEmail];
[self.delegate userDidLogin:user];
```

### Class Methods

Class methods (prefixed with `+`) are distinguished from instance methods (`-`):

```objc
+ (instancetype)sharedInstance;
- (void)instanceMethod;
```

### Forward Declarations

Forward declarations are tracked:

```objc
@class User;
@protocol UserDelegate;
```

### ARC vs Manual Memory Management

The parser handles both ARC and manual memory management code:

```objc
// ARC
@property (nonatomic, strong) NSString *name;

// Manual
- (void)dealloc {
    [_name release];
    [super dealloc];
}
```

## Performance

- **Average parsing speed**: ~60 files/second
- **Memory usage**: ~8MB per 1000 lines of code
- **Header pairing overhead**: ~10% additional time
- **Incremental parsing**: Supported via Tree-sitter

## Known Limitations

1. **Objective-C++**: Mixed Objective-C/C++ code (.mm files) may have limited C++ support
2. **Complex Macros**: Preprocessor macros with complex logic may not be fully analyzed
3. **Runtime Features**: Dynamic method resolution and swizzling cannot be statically analyzed

## Testing

The Objective-C parser includes comprehensive tests:

```bash
# Run Objective-C parser tests
go test ./internal/parser -run TestObjCParser

# Run header-implementation pairing tests
go test ./internal/parser -run TestObjCHeaderImpl

# Run with coverage
go test ./internal/parser -run TestObjCParser -cover
```

## Example Output

For the following Objective-C files:

**Person.h**:
```objc
#import <Foundation/Foundation.h>

@interface Person : NSObject
@property (nonatomic, strong) NSString *name;
- (void)greet;
@end
```

**Person.m**:
```objc
#import "Person.h"

@implementation Person
- (void)greet {
    NSLog(@"Hello, %@", self.name);
}
@end
```

**Parsed Output**:

```json
{
  "files": [
    {
      "path": "Person.h",
      "language": "objc",
      "symbols": [
        {
          "name": "Person",
          "kind": "interface",
          "line": 3
        },
        {
          "name": "name",
          "kind": "property",
          "line": 4
        },
        {
          "name": "greet",
          "kind": "method_declaration",
          "line": 5
        }
      ]
    },
    {
      "path": "Person.m",
      "language": "objc",
      "symbols": [
        {
          "name": "Person",
          "kind": "implementation",
          "line": 3
        },
        {
          "name": "greet",
          "kind": "method_implementation",
          "line": 4
        }
      ],
      "dependencies": [
        {
          "type": "implements_header",
          "source": "Person.m",
          "target": "Person.h"
        },
        {
          "type": "implements_declaration",
          "source": "greet (impl)",
          "target": "greet (decl)"
        }
      ]
    }
  ]
}
```

## Related Documentation

- [Swift Parser](swift-parser.md) - For Swift interop
- [C Parser](c-parser.md) - For C interop
- [Header-Implementation Association](header-implementation-association.md) - Detailed pairing algorithm
- [Parser Overview](README.md) - General parser documentation
