/**
 * @file functions.c
 * @brief Implementation of basic operations
 */

#include "functions.h"
#include <string.h>

int add(int a, int b) {
    return a + b;
}

int multiply(int a, int b) {
    int result = 0;
    for (int i = 0; i < b; i++) {
        result = add(result, a);
    }
    return result;
}

void greet(const char *name) {
    printf("Hello, %s!\n", name);
}

static int helper_function(int x) {
    return x * 2;
}
