/**
 * @file c_utilities.h
 * @brief C utilities to be called from Objective-C
 */

#ifndef C_UTILITIES_H
#define C_UTILITIES_H

#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief C struct definition
 */
struct c_data_t {
    int id;
    char name[256];
    double value;
    bool processed;
};

/**
 * @brief String to upper case
 */
void c_string_to_upper(char* str);

/**
 * @brief Validate string
 */
bool c_validate_string(const char* str);

/**
 * @brief Add two numbers
 */
double c_add(double a, double b);

/**
 * @brief Multiply two numbers
 */
double c_multiply(double a, double b);

/**
 * @brief Complex calculation
 */
double c_complex_calculation(double x, double y);

/**
 * @brief Initialize data struct
 */
void c_init_data(struct c_data_t* data);

/**
 * @brief Process data struct
 */
void c_process_data(struct c_data_t* data);

/**
 * @brief Validate data struct
 */
bool c_validate_data(const struct c_data_t* data);

/**
 * @brief Cleanup data struct
 */
void c_cleanup_data(struct c_data_t* data);

/**
 * @brief Log file operation
 */
void c_log_file_operation(const char* path, const char* operation);

/**
 * @brief Log message
 */
void c_log_message(const char* msg);

/**
 * @brief Process item
 */
void c_process_item(const char* item);

/**
 * @brief Cleanup all
 */
void c_cleanup_all(void);

/**
 * @brief Compress data
 */
unsigned char* c_compress_data(const unsigned char* data, size_t length, size_t* compressed_size);

/**
 * @brief Decompress data
 */
unsigned char* c_decompress_data(const unsigned char* data, size_t length, size_t* decompressed_size);

/**
 * @brief Calculate checksum
 */
unsigned int c_calculate_checksum(const unsigned char* data, size_t length);

#ifdef __cplusplus
}
#endif

#endif // C_UTILITIES_H
