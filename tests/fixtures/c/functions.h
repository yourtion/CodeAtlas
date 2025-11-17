/**
 * @file functions.h
 * @brief Function declarations for basic operations
 */

#ifndef FUNCTIONS_H
#define FUNCTIONS_H

#include <stdio.h>

/**
 * @brief Adds two integers
 * @param a First integer
 * @param b Second integer
 * @return Sum of a and b
 */
int add(int a, int b);

/**
 * @brief Multiplies two integers
 * @param a First integer
 * @param b Second integer
 * @return Product of a and b
 */
int multiply(int a, int b);

/**
 * @brief Prints a greeting message
 * @param name Name to greet
 */
void greet(const char *name);

#endif // FUNCTIONS_H
