/**
 * @file cpp_library.hpp
 * @brief C++ library to be called from Objective-C++
 */

#ifndef CPP_LIBRARY_HPP
#define CPP_LIBRARY_HPP

#include <string>
#include <vector>
#include <memory>
#include <algorithm>

namespace CppLibrary {

/**
 * @brief C++ class for data processing
 */
class DataProcessor {
public:
    DataProcessor();
    ~DataProcessor();
    
    /**
     * @brief Initialize processor
     */
    void initialize();
    
    /**
     * @brief Process data
     */
    bool process(const std::string& data);
    
    /**
     * @brief Sort data
     */
    void sortData(std::vector<double>& data);
    
    /**
     * @brief Get processed data
     */
    std::vector<std::string> getProcessedData() const;
    
    /**
     * @brief Cleanup
     */
    void cleanup();

private:
    std::vector<std::string> processedData_;
    bool initialized_;
};

/**
 * @brief Template function to calculate sum
 */
template<typename T>
T calculateSum(const std::vector<T>& values) {
    T sum = T();
    for (const auto& val : values) {
        sum += val;
    }
    return sum;
}

/**
 * @brief Log message
 */
template<typename T>
void logMessage(const T& message) {
    // Implementation
}

/**
 * @brief Validate input
 */
bool validateInput(const std::string& input);

/**
 * @brief Transform data
 */
std::string transformData(const std::string& input);

/**
 * @brief Create processor with smart pointer
 */
std::unique_ptr<DataProcessor> createProcessor();

/**
 * @brief Split string
 */
std::vector<std::string> split(const std::string& str, const std::string& delimiter);

} // namespace CppLibrary

#endif // CPP_LIBRARY_HPP
