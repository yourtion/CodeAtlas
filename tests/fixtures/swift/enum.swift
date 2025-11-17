import Foundation

/// Simple enum
enum Direction {
    case north
    case south
    case east
    case west
}

/// Enum with associated values
enum Result {
    case success(String)
    case failure(Error)
    case pending
}

/// Enum with raw values
enum StatusCode: Int {
    case ok = 200
    case notFound = 404
    case serverError = 500
}

/// Enum with methods
enum TrafficLight {
    case red
    case yellow
    case green
    
    /// Method on enum
    func canGo() -> Bool {
        switch self {
        case .green:
            return true
        default:
            return false
        }
    }
    
    /// Computed property
    var duration: Int {
        switch self {
        case .red:
            return 30
        case .yellow:
            return 5
        case .green:
            return 25
        }
    }
}

/// Enum conforming to protocol
enum Animal: CustomStringConvertible {
    case dog(name: String)
    case cat(name: String)
    case bird(name: String)
    
    var description: String {
        switch self {
        case .dog(let name):
            return "Dog named \(name)"
        case .cat(let name):
            return "Cat named \(name)"
        case .bird(let name):
            return "Bird named \(name)"
        }
    }
}
