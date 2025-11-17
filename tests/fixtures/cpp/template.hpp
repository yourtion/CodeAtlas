/**
 * @file template.hpp
 * @brief Example C++ template file
 */

#ifndef TEMPLATE_HPP
#define TEMPLATE_HPP

#include <iostream>
#include <vector>

namespace example {

/**
 * @brief Template class for a simple container
 * @tparam T The type of elements to store
 */
template<typename T>
class Container {
public:
    /**
     * @brief Add an element
     * @param element The element to add
     */
    void add(const T& element) {
        elements_.push_back(element);
    }
    
    /**
     * @brief Get element at index
     * @param index The index
     * @return The element
     */
    T get(size_t index) const {
        return elements_[index];
    }
    
    /**
     * @brief Get the size
     * @return The number of elements
     */
    size_t size() const {
        return elements_.size();
    }
    
    /**
     * @brief Print all elements
     */
    void print() const {
        for (const auto& elem : elements_) {
            std::cout << elem << " ";
        }
        std::cout << std::endl;
    }

private:
    std::vector<T> elements_;
};

/**
 * @brief Template function to find maximum
 * @tparam T The type of values
 * @param a First value
 * @param b Second value
 * @return The maximum value
 */
template<typename T>
T max(const T& a, const T& b) {
    return (a > b) ? a : b;
}

/**
 * @brief Template specialization for strings
 */
template<>
class Container<std::string> {
public:
    void add(const std::string& element) {
        elements_.push_back(element);
    }
    
    std::string get(size_t index) const {
        return elements_[index];
    }
    
    size_t size() const {
        return elements_.size();
    }
    
    void print() const {
        for (const auto& elem : elements_) {
            std::cout << "\"" << elem << "\" ";
        }
        std::cout << std::endl;
    }

private:
    std::vector<std::string> elements_;
};

/**
 * @brief Template class with multiple type parameters
 * @tparam K Key type
 * @tparam V Value type
 */
template<typename K, typename V>
class Pair {
public:
    Pair(const K& key, const V& value) : key_(key), value_(value) {}
    
    K getKey() const { return key_; }
    V getValue() const { return value_; }
    
    void setKey(const K& key) { key_ = key; }
    void setValue(const V& value) { value_ = value; }

private:
    K key_;
    V value_;
};

} // namespace example

#endif // TEMPLATE_HPP
