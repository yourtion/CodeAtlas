/**
 * @file class.hpp
 * @brief Example C++ class header file
 */

#ifndef CLASS_HPP
#define CLASS_HPP

#include <string>
#include <vector>

namespace example {

/**
 * @brief A simple example class
 * 
 * This class demonstrates basic C++ features including
 * member variables, methods, and constructors.
 */
class MyClass {
public:
    /**
     * @brief Default constructor
     */
    MyClass();
    
    /**
     * @brief Constructor with parameters
     * @param name The name to set
     * @param value The initial value
     */
    MyClass(const std::string& name, int value);
    
    /**
     * @brief Destructor
     */
    ~MyClass();
    
    /**
     * @brief Get the name
     * @return The current name
     */
    std::string getName() const;
    
    /**
     * @brief Set the name
     * @param name The new name
     */
    void setName(const std::string& name);
    
    /**
     * @brief Get the value
     * @return The current value
     */
    int getValue() const;
    
    /**
     * @brief Process data
     * @param data The data to process
     * @return True if successful
     */
    bool processData(const std::vector<int>& data);
    
    /**
     * @brief Virtual method for polymorphism
     */
    virtual void virtualMethod();
    
    /**
     * @brief Pure virtual method
     */
    virtual void pureVirtualMethod() = 0;

private:
    std::string name_;
    int value_;
    std::vector<int> data_;
};

/**
 * @brief Derived class demonstrating inheritance
 */
class DerivedClass : public MyClass {
public:
    DerivedClass();
    
    /**
     * @brief Override virtual method
     */
    void virtualMethod() override;
    
    /**
     * @brief Implement pure virtual method
     */
    void pureVirtualMethod() override;
    
    /**
     * @brief Additional method in derived class
     */
    void derivedMethod();
};

} // namespace example

#endif // CLASS_HPP
