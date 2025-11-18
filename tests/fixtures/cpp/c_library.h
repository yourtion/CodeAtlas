/**
 * @file c_library.h
 * @brief C library header to be called from C++
 */

#ifndef C_LIBRARY_H
#define C_LIBRARY_H

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief C struct definition
 */
struct c_data_t {
    int value;
    char name[256];
    void* ptr;
};

/**
 * @brief Initialize C library
 */
void c_init(void);

/**
 * @brief Cleanup C library
 */
void c_cleanup(void);

/**
 * @brief Free memory
 */
void c_free(void* ptr);

/**
 * @brief Process string
 */
int c_process_string(const char* str);

/**
 * @brief Add two numbers
 */
double c_add(double a, double b);

/**
 * @brief Multiply two numbers
 */
double c_multiply(double a, double b);

/**
 * @brief Initialize struct
 */
void c_init_struct(struct c_data_t* data);

/**
 * @brief Process struct
 */
void c_process_struct(struct c_data_t* data);

/**
 * @brief Free struct
 */
void c_free_struct(struct c_data_t* data);

/**
 * @brief Log message
 */
void c_log_message(const char* msg);

/**
 * @brief Validate input
 */
int c_validate_input(const char* input);

#ifdef __cplusplus
}
#endif

#endif // C_LIBRARY_H
