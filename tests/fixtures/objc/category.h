//
//  NSString+Utilities.h
//  Example
//
//  Category on NSString
//

#import <Foundation/Foundation.h>

/**
 * Utility methods for NSString
 */
@interface NSString (Utilities)

/// Check if string is empty
- (BOOL)isEmpty;

/// Reverse the string
- (NSString *)reverse;

/// Convert to uppercase with locale
- (NSString *)uppercaseStringWithLocale:(NSLocale *)locale;

@end
