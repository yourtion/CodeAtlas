/**
 * @file macros.h
 * @brief Example of preprocessor macros
 */

#ifndef MACROS_H
#define MACROS_H

/// Maximum buffer size
#define MAX_BUFFER_SIZE 1024

/// Minimum value
#define MIN_VALUE 0

/**
 * @brief Macro to find maximum of two values
 */
#define MAX(a, b) ((a) > (b) ? (a) : (b))

/**
 * @brief Macro to find minimum of two values
 */
#define MIN(a, b) ((a) < (b) ? (a) : (b))

/**
 * @brief Macro to square a number
 */
#define SQUARE(x) ((x) * (x))

// Global variable declaration
extern int global_counter;

#endif // MACROS_H
