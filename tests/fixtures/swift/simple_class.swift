import Foundation

/// A simple user class demonstrating Swift class features
class User {
    /// The user's name
    var name: String
    
    /// The user's age
    var age: Int
    
    /// Computed property for display name
    var displayName: String {
        return "User: \(name)"
    }
    
    /// Property with observers
    var status: String = "active" {
        willSet {
            print("Status will change to \(newValue)")
        }
        didSet {
            print("Status changed from \(oldValue)")
        }
    }
    
    /// Initializer
    init(name: String, age: Int) {
        self.name = name
        self.age = age
    }
    
    /// Greet method
    func greet() -> String {
        return "Hello, I'm \(name)"
    }
    
    /// Update age method
    func updateAge(_ newAge: Int) {
        self.age = newAge
    }
}

/// Subclass demonstrating inheritance
class AdminUser: User {
    var permissions: [String]
    
    init(name: String, age: Int, permissions: [String]) {
        self.permissions = permissions
        super.init(name: name, age: age)
    }
    
    override func greet() -> String {
        return "Hello, I'm admin \(name)"
    }
}
