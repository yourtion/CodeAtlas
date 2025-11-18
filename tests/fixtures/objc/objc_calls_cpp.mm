/**
 * @file objc_calls_cpp.mm
 * @brief Objective-C++ code that calls C++ classes and functions
 */

#import <Foundation/Foundation.h>
#include <string>
#include <vector>

// Include C++ header
#include "cpp_library.hpp"

/**
 * @brief Objective-C class that wraps C++ functionality
 */
@interface CppWrapper : NSObject {
    // C++ object as instance variable
    CppLibrary::DataProcessor* processor;
    std::vector<std::string>* dataList;
}

/**
 * @brief Initialize with C++ objects
 */
- (instancetype)init;

/**
 * @brief Process data using C++ class
 */
- (BOOL)processData:(NSString*)input;

/**
 * @brief Calculate using C++ template function
 */
- (double)calculateSum:(NSArray*)numbers;

/**
 * @brief Use C++ STL containers
 */
- (NSArray*)getProcessedData;

/**
 * @brief Cleanup
 */
- (void)dealloc;

@end

@implementation CppWrapper

- (instancetype)init {
    self = [super init];
    if (self) {
        // Create C++ objects
        processor = new CppLibrary::DataProcessor();
        dataList = new std::vector<std::string>();
        
        // Call C++ method
        processor->initialize();
    }
    return self;
}

- (BOOL)processData:(NSString*)input {
    // Convert NSString to C++ string
    std::string cppString = [input UTF8String];
    
    // Call C++ method
    bool result = processor->process(cppString);
    
    // Add to C++ vector
    dataList->push_back(cppString);
    
    // Call C++ template function
    CppLibrary::logMessage(cppString);
    
    return result ? YES : NO;
}

- (double)calculateSum:(NSArray*)numbers {
    // Create C++ vector
    std::vector<double> cppNumbers;
    
    // Convert NSArray to C++ vector
    for (NSNumber* num in numbers) {
        cppNumbers.push_back([num doubleValue]);
    }
    
    // Call C++ template function
    double sum = CppLibrary::calculateSum(cppNumbers);
    
    // Call C++ algorithm
    processor->sortData(cppNumbers);
    
    return sum;
}

- (NSArray*)getProcessedData {
    // Get data from C++ vector
    NSMutableArray* result = [NSMutableArray array];
    
    for (const auto& str : *dataList) {
        NSString* nsStr = [NSString stringWithUTF8String:str.c_str()];
        [result addObject:nsStr];
    }
    
    // Call C++ method to get additional data
    std::vector<std::string> processed = processor->getProcessedData();
    
    for (const auto& str : processed) {
        NSString* nsStr = [NSString stringWithUTF8String:str.c_str()];
        [result addObject:nsStr];
    }
    
    return result;
}

- (void)dealloc {
    // Call C++ cleanup
    processor->cleanup();
    
    // Delete C++ objects
    delete processor;
    delete dataList;
}

@end

/**
 * @brief Objective-C function that calls C++ free functions
 */
void processWithCpp(NSString* input) {
    // Convert to C++ string
    std::string cppStr = [input UTF8String];
    
    // Call C++ free functions
    CppLibrary::validateInput(cppStr);
    CppLibrary::transformData(cppStr);
    
    // Use C++ smart pointers
    auto ptr = CppLibrary::createProcessor();
    ptr->process(cppStr);
}

/**
 * @brief Objective-C category that uses C++ STL
 */
@interface NSString (CppExtensions)
- (NSString*)reverseUsingCpp;
- (NSArray*)splitUsingCpp:(NSString*)delimiter;
@end

@implementation NSString (CppExtensions)

- (NSString*)reverseUsingCpp {
    // Convert to C++ string
    std::string cppStr = [self UTF8String];
    
    // Use C++ algorithm
    std::reverse(cppStr.begin(), cppStr.end());
    
    // Convert back to NSString
    return [NSString stringWithUTF8String:cppStr.c_str()];
}

- (NSArray*)splitUsingCpp:(NSString*)delimiter {
    // Use C++ string operations
    std::string cppStr = [self UTF8String];
    std::string delim = [delimiter UTF8String];
    
    // Call C++ split function
    std::vector<std::string> parts = CppLibrary::split(cppStr, delim);
    
    // Convert to NSArray
    NSMutableArray* result = [NSMutableArray array];
    for (const auto& part : parts) {
        [result addObject:[NSString stringWithUTF8String:part.c_str()]];
    }
    
    return result;
}

@end

/**
 * @brief Main function demonstrating Objective-C calling C++
 */
int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // Use Objective-C wrapper for C++
        CppWrapper* wrapper = [[CppWrapper alloc] init];
        [wrapper processData:@"test data"];
        
        NSArray* numbers = @[@1.0, @2.0, @3.0, @4.0];
        double sum = [wrapper calculateSum:numbers];
        NSLog(@"Sum: %f", sum);
        
        NSArray* data = [wrapper getProcessedData];
        NSLog(@"Processed data: %@", data);
        
        // Direct C++ function calls
        processWithCpp(@"direct call");
        
        // Use category methods
        NSString* str = @"Hello";
        NSString* reversed = [str reverseUsingCpp];
        NSArray* parts = [str splitUsingCpp:@","];
    }
    return 0;
}
