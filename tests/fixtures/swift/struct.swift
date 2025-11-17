import Foundation

/// Simple struct demonstrating Swift struct features
struct Point {
    var x: Double
    var y: Double
    
    /// Computed property
    var magnitude: Double {
        return (x * x + y * y).squareRoot()
    }
    
    /// Method
    func distance(to other: Point) -> Double {
        let dx = x - other.x
        let dy = y - other.y
        return (dx * dx + dy * dy).squareRoot()
    }
    
    /// Mutating method
    mutating func move(dx: Double, dy: Double) {
        x += dx
        y += dy
    }
}

/// Struct with nested type
struct Game {
    var score: Int
    
    /// Nested enum
    enum Difficulty {
        case easy
        case medium
        case hard
    }
    
    var difficulty: Difficulty
    
    init(difficulty: Difficulty) {
        self.score = 0
        self.difficulty = difficulty
    }
}

/// Generic struct
struct Stack<Element> {
    private var items: [Element] = []
    
    mutating func push(_ item: Element) {
        items.append(item)
    }
    
    mutating func pop() -> Element? {
        return items.popLast()
    }
    
    var count: Int {
        return items.count
    }
}
