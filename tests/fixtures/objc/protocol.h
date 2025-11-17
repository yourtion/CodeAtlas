//
//  MyProtocol.h
//  Example
//
//  Protocol definition example
//

#import <Foundation/Foundation.h>

/**
 * A protocol defining data source methods
 */
@protocol DataSource <NSObject>

@required
/// Get the number of items
- (NSInteger)numberOfItems;

/// Get item at index
- (id)itemAtIndex:(NSInteger)index;

@optional
/// Optional method to refresh data
- (void)refreshData;

@end

/**
 * A class that conforms to the protocol
 */
@interface DataProvider : NSObject <DataSource>

@property (nonatomic, strong) NSArray *items;

- (instancetype)initWithItems:(NSArray *)items;

@end
