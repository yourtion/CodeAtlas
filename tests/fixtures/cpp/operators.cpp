/**
 * @file operators.cpp
 * @brief Example C++ file with operator overloading
 */

#include <iostream>

/**
 * @brief Complex number class with operator overloading
 */
class Complex {
public:
    Complex(double real = 0.0, double imag = 0.0) 
        : real_(real), imag_(imag) {}
    
    // Arithmetic operators
    Complex operator+(const Complex& other) const {
        return Complex(real_ + other.real_, imag_ + other.imag_);
    }
    
    Complex operator-(const Complex& other) const {
        return Complex(real_ - other.real_, imag_ - other.imag_);
    }
    
    Complex operator*(const Complex& other) const {
        return Complex(
            real_ * other.real_ - imag_ * other.imag_,
            real_ * other.imag_ + imag_ * other.real_
        );
    }
    
    // Comparison operators
    bool operator==(const Complex& other) const {
        return real_ == other.real_ && imag_ == other.imag_;
    }
    
    bool operator!=(const Complex& other) const {
        return !(*this == other);
    }
    
    // Assignment operator
    Complex& operator=(const Complex& other) {
        if (this != &other) {
            real_ = other.real_;
            imag_ = other.imag_;
        }
        return *this;
    }
    
    // Compound assignment operators
    Complex& operator+=(const Complex& other) {
        real_ += other.real_;
        imag_ += other.imag_;
        return *this;
    }
    
    // Unary operators
    Complex operator-() const {
        return Complex(-real_, -imag_);
    }
    
    // Increment/decrement operators
    Complex& operator++() {
        ++real_;
        return *this;
    }
    
    Complex operator++(int) {
        Complex temp = *this;
        ++real_;
        return temp;
    }
    
    // Stream operators (friend functions)
    friend std::ostream& operator<<(std::ostream& os, const Complex& c);
    friend std::istream& operator>>(std::istream& is, Complex& c);
    
    // Subscript operator
    double& operator[](int index) {
        return (index == 0) ? real_ : imag_;
    }
    
    // Function call operator
    double operator()() const {
        return real_ * real_ + imag_ * imag_;
    }

private:
    double real_;
    double imag_;
};

// Friend function implementations
std::ostream& operator<<(std::ostream& os, const Complex& c) {
    os << c.real_ << " + " << c.imag_ << "i";
    return os;
}

std::istream& operator>>(std::istream& is, Complex& c) {
    is >> c.real_ >> c.imag_;
    return is;
}

int main() {
    Complex c1(3.0, 4.0);
    Complex c2(1.0, 2.0);
    
    // Use operators
    Complex c3 = c1 + c2;
    Complex c4 = c1 - c2;
    Complex c5 = c1 * c2;
    
    bool equal = (c1 == c2);
    bool notEqual = (c1 != c2);
    
    c1 += c2;
    ++c1;
    
    std::cout << c1 << std::endl;
    
    double magnitude = c1();
    
    return 0;
}
