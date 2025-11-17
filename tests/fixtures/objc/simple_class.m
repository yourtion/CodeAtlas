//
//  MyClass.m
//  Example
//
//  Simple Objective-C class implementation
//

#import "simple_class.h"

@implementation MyClass

- (instancetype)initWithName:(NSString *)name age:(NSInteger)age {
    self = [super init];
    if (self) {
        _name = name;
        _age = age;
    }
    return self;
}

- (NSString *)greet {
    return [NSString stringWithFormat:@"Hello, I'm %@ and I'm %ld years old", 
            self.name, (long)self.age];
}

+ (instancetype)personWithName:(NSString *)name {
    return [[MyClass alloc] initWithName:name age:0];
}

- (void)privateMethod {
    NSLog(@"This is a private method");
}

@end
