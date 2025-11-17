import Foundation

/// Base class
class Person {
    var name: String
    
    init(name: String) {
        self.name = name
    }
}

/// Extension adding functionality
extension Person {
    /// Computed property added via extension
    var initials: String {
        let components = name.split(separator: " ")
        return components.map { String($0.prefix(1)) }.joined()
    }
    
    /// Method added via extension
    func introduce() -> String {
        return "My name is \(name)"
    }
}

/// Extension adding protocol conformance
extension Person: CustomStringConvertible {
    var description: String {
        return "Person(name: \(name))"
    }
}

/// Extension on built-in type
extension String {
    /// Check if string is email
    func isEmail() -> Bool {
        return self.contains("@")
    }
    
    /// Reverse string
    func reversed() -> String {
        return String(self.reversed())
    }
}
