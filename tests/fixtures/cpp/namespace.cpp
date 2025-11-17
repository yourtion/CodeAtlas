/**
 * @file namespace.cpp
 * @brief Example C++ file with namespaces
 */

#include <iostream>
#include <string>

/**
 * @brief Outer namespace
 */
namespace outer {

/**
 * @brief Inner namespace
 */
namespace inner {

/**
 * @brief Function in nested namespace
 * @param message The message to print
 */
void printMessage(const std::string& message) {
    std::cout << "Inner: " << message << std::endl;
}

/**
 * @brief Class in nested namespace
 */
class InnerClass {
public:
    InnerClass() : value_(0) {}
    
    void setValue(int value) {
        value_ = value;
    }
    
    int getValue() const {
        return value_;
    }

private:
    int value_;
};

} // namespace inner

/**
 * @brief Function in outer namespace
 * @param x First number
 * @param y Second number
 * @return The sum
 */
int add(int x, int y) {
    return x + y;
}

/**
 * @brief Class in outer namespace
 */
class OuterClass {
public:
    OuterClass() : name_("") {}
    
    void setName(const std::string& name) {
        name_ = name;
    }
    
    std::string getName() const {
        return name_;
    }
    
    void useInner() {
        // Use inner namespace
        inner::printMessage("Hello from outer");
        
        inner::InnerClass obj;
        obj.setValue(42);
        int val = obj.getValue();
    }

private:
    std::string name_;
};

} // namespace outer

/**
 * @brief Global namespace function
 */
void globalFunction() {
    std::cout << "Global function" << std::endl;
    
    // Use outer namespace
    int result = outer::add(10, 20);
    
    // Use nested namespace
    outer::inner::printMessage("Hello from global");
    
    // Create objects
    outer::OuterClass outerObj;
    outerObj.setName("Outer");
    
    outer::inner::InnerClass innerObj;
    innerObj.setValue(100);
}

/**
 * @brief Main function
 */
int main() {
    globalFunction();
    
    // Using namespace
    using namespace outer;
    OuterClass obj;
    obj.setName("Test");
    
    // Using specific function
    using outer::inner::printMessage;
    printMessage("Direct call");
    
    return 0;
}
