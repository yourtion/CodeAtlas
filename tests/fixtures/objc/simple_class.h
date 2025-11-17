//
//  MyClass.h
//  Example
//
//  Simple Objective-C class header
//

#import <Foundation/Foundation.h>

/**
 * A simple example class demonstrating Objective-C features
 */
@interface MyClass : NSObject

/// The name property
@property (nonatomic, strong) NSString *name;

/// The age property
@property (nonatomic, assign) NSInteger age;

/**
 * Initialize with name and age
 * @param name The person's name
 * @param age The person's age
 * @return An initialized instance
 */
- (instancetype)initWithName:(NSString *)name age:(NSInteger)age;

/**
 * Get a greeting message
 * @return A greeting string
 */
- (NSString *)greet;

/**
 * Class method to create an instance
 * @param name The person's name
 * @return A new instance
 */
+ (instancetype)personWithName:(NSString *)name;

@end
