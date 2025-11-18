/**
 * @file simple_cpp_calls.mm
 * @brief Simple Objective-C++ code that calls C++ functions
 */

#import <Foundation/Foundation.h>
#include <string>
#include <vector>

// Simple C++ class
class CppHelper {
public:
    static int add(int a, int b) {
        return a + b;
    }
    
    static std::string getMessage() {
        return "Hello from C++";
    }
};

// Simple C++ function
int cpp_multiply(int a, int b) {
    return a * b;
}

/**
 * @brief Objective-C class that uses C++ code
 */
@interface CppBridge : NSObject

- (int)addNumbers:(int)a and:(int)b;
- (NSString*)getCppMessage;
- (int)multiplyNumbers:(int)a and:(int)b;

@end

@implementation CppBridge

- (int)addNumbers:(int)a and:(int)b {
    // Call C++ static method
    return CppHelper::add(a, b);
}

- (NSString*)getCppMessage {
    // Call C++ static method that returns std::string
    std::string cppStr = CppHelper::getMessage();
    
    // Convert to NSString
    return [NSString stringWithUTF8String:cppStr.c_str()];
}

- (int)multiplyNumbers:(int)a and:(int)b {
    // Call C++ function
    return cpp_multiply(a, b);
}

@end

// Function that uses C++ STL
void processWithCpp(NSArray* items) {
    // Use C++ vector
    std::vector<std::string> cppVector;
    
    for (NSString* item in items) {
        std::string cppStr = [item UTF8String];
        cppVector.push_back(cppStr);
    }
    
    // Use C++ algorithm
    for (const auto& str : cppVector) {
        NSLog(@"Item: %s", str.c_str());
    }
}
