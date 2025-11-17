/**
 * @file class.cpp
 * @brief Implementation of MyClass
 */

#include "class.hpp"
#include <algorithm>
#include <iostream>

namespace example {

// Default constructor
MyClass::MyClass() : name_(""), value_(0) {
    std::cout << "MyClass default constructor" << std::endl;
}

// Constructor with parameters
MyClass::MyClass(const std::string& name, int value) 
    : name_(name), value_(value) {
    std::cout << "MyClass parameterized constructor" << std::endl;
}

// Destructor
MyClass::~MyClass() {
    std::cout << "MyClass destructor" << std::endl;
}

// Get the name
std::string MyClass::getName() const {
    return name_;
}

// Set the name
void MyClass::setName(const std::string& name) {
    name_ = name;
}

// Get the value
int MyClass::getValue() const {
    return value_;
}

// Process data
bool MyClass::processData(const std::vector<int>& data) {
    data_ = data;
    
    // Sort the data
    std::sort(data_.begin(), data_.end());
    
    // Call helper function
    helperFunction();
    
    return !data_.empty();
}

// Virtual method
void MyClass::virtualMethod() {
    std::cout << "MyClass::virtualMethod" << std::endl;
}

// Helper function (not in header)
static void helperFunction() {
    std::cout << "Helper function" << std::endl;
}

// DerivedClass implementation

DerivedClass::DerivedClass() : MyClass() {
    std::cout << "DerivedClass constructor" << std::endl;
}

void DerivedClass::virtualMethod() {
    std::cout << "DerivedClass::virtualMethod" << std::endl;
    MyClass::virtualMethod(); // Call base class method
}

void DerivedClass::pureVirtualMethod() {
    std::cout << "DerivedClass::pureVirtualMethod" << std::endl;
}

void DerivedClass::derivedMethod() {
    std::cout << "DerivedClass::derivedMethod" << std::endl;
    
    // Call inherited methods
    setName("Derived");
    std::string name = getName();
}

} // namespace example
