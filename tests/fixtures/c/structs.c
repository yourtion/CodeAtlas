/**
 * @file structs.c
 * @brief Example of struct and union declarations
 */

#include <stdio.h>

/**
 * @brief Represents a point in 2D space
 */
struct Point {
    int x;
    int y;
};

/**
 * @brief Represents a person
 */
struct Person {
    char name[50];
    int age;
    struct Point location;
};

/**
 * @brief Union for storing different data types
 */
union Data {
    int i;
    float f;
    char str[20];
};

/**
 * @brief Enum for days of the week
 */
enum Day {
    MONDAY,
    TUESDAY,
    WEDNESDAY,
    THURSDAY,
    FRIDAY,
    SATURDAY,
    SUNDAY
};

typedef struct Point Point;
typedef enum Day Day;

void print_point(Point p) {
    printf("Point(%d, %d)\n", p.x, p.y);
}
