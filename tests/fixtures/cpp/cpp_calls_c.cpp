/**
 * @file cpp_calls_c.cpp
 * @brief C++ code that calls C functions
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Include C header
extern "C" {
    #include "c_library.h"
}

namespace wrapper {

/**
 * @brief C++ wrapper class that uses C functions
 */
class CWrapper {
public:
    CWrapper() {
        // Call C function
        c_init();
        data_ = nullptr;
    }
    
    ~CWrapper() {
        // Call C cleanup function
        if (data_) {
            c_free(data_);
        }
        c_cleanup();
    }
    
    /**
     * @brief Process data using C functions
     */
    bool processData(const char* input) {
        // Call C string function
        size_t len = strlen(input);
        
        // Call C memory allocation
        data_ = (char*)malloc(len + 1);
        if (!data_) {
            return false;
        }
        
        // Call C string copy
        strcpy(data_, input);
        
        // Call custom C function
        int result = c_process_string(data_);
        
        return result == 0;
    }
    
    /**
     * @brief Calculate using C math functions
     */
    double calculate(double x, double y) {
        // Call C math functions
        double sum = c_add(x, y);
        double product = c_multiply(x, y);
        
        return sum + product;
    }
    
    /**
     * @brief Use C struct
     */
    void useStruct() {
        // Create C struct
        struct c_data_t cdata;
        c_init_struct(&cdata);
        
        // Call C function with struct
        c_process_struct(&cdata);
        
        // Cleanup
        c_free_struct(&cdata);
    }

private:
    char* data_;
};

/**
 * @brief Free function that calls C functions
 */
void processCData(const char* input) {
    // Direct C function calls
    printf("Processing: %s\n", input);
    
    // Call custom C functions
    c_log_message(input);
    c_validate_input(input);
}

} // namespace wrapper

/**
 * @brief Main function demonstrating C++ calling C
 */
int main() {
    // Use C standard library functions
    printf("C++ calling C functions\n");
    
    // Use wrapper class
    wrapper::CWrapper wrapper;
    wrapper.processData("test data");
    wrapper.calculate(10.0, 20.0);
    wrapper.useStruct();
    
    // Direct C function calls
    wrapper::processCData("direct call");
    
    return 0;
}
