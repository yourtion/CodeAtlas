import Foundation

/// Protocol defining drawable behavior
protocol Drawable {
    /// Draw method requirement
    func draw()
    
    /// Color property requirement
    var color: String { get set }
}

/// Protocol defining resizable behavior
protocol Resizable {
    func resize(width: Int, height: Int)
}

/// Combined protocol
protocol Shape: Drawable, Resizable {
    var area: Double { get }
}

/// Class conforming to protocol
class Circle: Drawable {
    var color: String
    var radius: Double
    
    init(color: String, radius: Double) {
        self.color = color
        self.radius = radius
    }
    
    func draw() {
        print("Drawing circle with color \(color)")
    }
}

/// Struct conforming to multiple protocols
struct Rectangle: Shape {
    var color: String
    var width: Int
    var height: Int
    
    var area: Double {
        return Double(width * height)
    }
    
    func draw() {
        print("Drawing rectangle")
    }
    
    func resize(width: Int, height: Int) {
        // Resize logic
    }
}
