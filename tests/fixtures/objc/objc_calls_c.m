/**
 * @file objc_calls_c.m
 * @brief Objective-C code that calls C functions
 */

#import <Foundation/Foundation.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

// Include C header
#include "c_utilities.h"

/**
 * @brief Objective-C class that uses C functions
 */
@interface CUtilityWrapper : NSObject

/**
 * @brief Process string using C functions
 */
- (NSString*)processString:(NSString*)input;

/**
 * @brief Calculate using C math functions
 */
- (double)calculate:(double)x with:(double)y;

/**
 * @brief Use C struct
 */
- (NSDictionary*)processStruct:(NSDictionary*)input;

/**
 * @brief Call C file operations
 */
- (BOOL)writeToFile:(NSString*)path content:(NSString*)content;

@end

@implementation CUtilityWrapper

- (NSString*)processString:(NSString*)input {
    // Convert NSString to C string
    const char* cStr = [input UTF8String];
    
    // Call C string functions
    size_t len = strlen(cStr);
    char* buffer = (char*)malloc(len + 1);
    
    if (!buffer) {
        return nil;
    }
    
    // Call C string copy
    strcpy(buffer, cStr);
    
    // Call custom C function
    c_string_to_upper(buffer);
    
    // Call C validation function
    if (!c_validate_string(buffer)) {
        free(buffer);
        return nil;
    }
    
    // Convert back to NSString
    NSString* result = [NSString stringWithUTF8String:buffer];
    
    // Free C memory
    free(buffer);
    
    return result;
}

- (double)calculate:(double)x with:(double)y {
    // Call C math functions
    double sum = c_add(x, y);
    double product = c_multiply(x, y);
    double power = pow(x, y);
    
    // Call custom C math function
    double result = c_complex_calculation(sum, product);
    
    return result + power;
}

- (NSDictionary*)processStruct:(NSDictionary*)input {
    // Create C struct
    struct c_data_t data;
    
    // Initialize using C function
    c_init_data(&data);
    
    // Set values from dictionary
    data.id = [[input objectForKey:@"id"] intValue];
    const char* name = [[input objectForKey:@"name"] UTF8String];
    strncpy(data.name, name, sizeof(data.name) - 1);
    data.value = [[input objectForKey:@"value"] doubleValue];
    
    // Process using C function
    c_process_data(&data);
    
    // Call C validation
    if (!c_validate_data(&data)) {
        return nil;
    }
    
    // Convert back to dictionary
    NSDictionary* result = @{
        @"id": @(data.id),
        @"name": [NSString stringWithUTF8String:data.name],
        @"value": @(data.value),
        @"processed": @(data.processed)
    };
    
    // Cleanup using C function
    c_cleanup_data(&data);
    
    return result;
}

- (BOOL)writeToFile:(NSString*)path content:(NSString*)content {
    // Convert to C strings
    const char* cPath = [path UTF8String];
    const char* cContent = [content UTF8String];
    
    // Call C file operations
    FILE* file = fopen(cPath, "w");
    if (!file) {
        return NO;
    }
    
    // Write using C function
    size_t written = fwrite(cContent, 1, strlen(cContent), file);
    
    // Close using C function
    fclose(file);
    
    // Call custom C logging function
    c_log_file_operation(cPath, "write");
    
    return written > 0;
}

@end

/**
 * @brief Objective-C function that calls C functions
 */
void processDataWithC(NSArray* items) {
    // Use C standard library
    printf("Processing %lu items\n", (unsigned long)[items count]);
    
    for (NSString* item in items) {
        const char* cStr = [item UTF8String];
        
        // Call C functions
        c_log_message(cStr);
        c_process_item(cStr);
    }
    
    // Call C cleanup
    c_cleanup_all();
}

/**
 * @brief Objective-C category using C functions
 */
@interface NSData (CExtensions)
- (NSData*)compressUsingC;
- (NSData*)decompressUsingC;
- (NSString*)checksumUsingC;
@end

@implementation NSData (CExtensions)

- (NSData*)compressUsingC {
    // Get C buffer
    const unsigned char* bytes = (const unsigned char*)[self bytes];
    size_t length = [self length];
    
    // Call C compression function
    size_t compressedSize;
    unsigned char* compressed = c_compress_data(bytes, length, &compressedSize);
    
    if (!compressed) {
        return nil;
    }
    
    // Create NSData
    NSData* result = [NSData dataWithBytes:compressed length:compressedSize];
    
    // Free C memory
    free(compressed);
    
    return result;
}

- (NSData*)decompressUsingC {
    // Get C buffer
    const unsigned char* bytes = (const unsigned char*)[self bytes];
    size_t length = [self length];
    
    // Call C decompression function
    size_t decompressedSize;
    unsigned char* decompressed = c_decompress_data(bytes, length, &decompressedSize);
    
    if (!decompressed) {
        return nil;
    }
    
    // Create NSData
    NSData* result = [NSData dataWithBytes:decompressed length:decompressedSize];
    
    // Free C memory
    free(decompressed);
    
    return result;
}

- (NSString*)checksumUsingC {
    // Get C buffer
    const unsigned char* bytes = (const unsigned char*)[self bytes];
    size_t length = [self length];
    
    // Call C checksum function
    unsigned int checksum = c_calculate_checksum(bytes, length);
    
    // Convert to string
    return [NSString stringWithFormat:@"%08X", checksum];
}

@end

/**
 * @brief Main function demonstrating Objective-C calling C
 */
int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // Use wrapper class
        CUtilityWrapper* wrapper = [[CUtilityWrapper alloc] init];
        
        NSString* processed = [wrapper processString:@"hello world"];
        NSLog(@"Processed: %@", processed);
        
        double result = [wrapper calculate:10.0 with:5.0];
        NSLog(@"Result: %f", result);
        
        NSDictionary* input = @{@"id": @1, @"name": @"test", @"value": @42.0};
        NSDictionary* output = [wrapper processStruct:input];
        NSLog(@"Output: %@", output);
        
        // Direct C function calls
        NSArray* items = @[@"item1", @"item2", @"item3"];
        processDataWithC(items);
        
        // Use category methods
        NSData* data = [@"test data" dataUsingEncoding:NSUTF8StringEncoding];
        NSData* compressed = [data compressUsingC];
        NSString* checksum = [data checksumUsingC];
        NSLog(@"Checksum: %@", checksum);
    }
    return 0;
}
