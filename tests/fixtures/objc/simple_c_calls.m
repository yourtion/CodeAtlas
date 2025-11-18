/**
 * @file simple_c_calls.m
 * @brief Simple Objective-C code that calls C functions
 */

#import <Foundation/Foundation.h>
#include <stdio.h>
#include <string.h>

// Simple C function declarations
int c_add(int a, int b);
void c_log(const char* message);

/**
 * @brief Simple Objective-C class that calls C functions
 */
@interface SimpleWrapper : NSObject

- (int)addNumbers:(int)a and:(int)b;
- (void)logMessage:(NSString*)message;

@end

@implementation SimpleWrapper

- (int)addNumbers:(int)a and:(int)b {
    // Call C function
    return c_add(a, b);
}

- (void)logMessage:(NSString*)message {
    // Convert to C string and call C function
    const char* cStr = [message UTF8String];
    c_log(cStr);
    
    // Also call standard C library
    printf("Message: %s\n", cStr);
    
    // Call strlen
    size_t len = strlen(cStr);
    printf("Length: %zu\n", len);
}

@end

// Simple function that calls C
void processWithC(NSString* input) {
    const char* cStr = [input UTF8String];
    c_log(cStr);
    printf("Processing: %s\n", cStr);
}
