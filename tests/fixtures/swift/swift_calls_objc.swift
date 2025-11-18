// Swift calling Objective-C code
// This demonstrates Swift-ObjC interop in iOS/macOS development

import Foundation
import UIKit

// Using Objective-C class from Swift
class SwiftViewController: UIViewController {
    
    // Using NSString (Objective-C class)
    func processString() {
        let objcString = NSString(string: "Hello from ObjC")
        let length = objcString.length
        let uppercased = objcString.uppercased
        print("Length: \(length), Uppercased: \(uppercased)")
    }
    
    // Using NSArray (Objective-C collection)
    func processArray() {
        let objcArray = NSArray(array: [1, 2, 3])
        let count = objcArray.count
        let firstObject = objcArray.firstObject
        print("Count: \(count), First: \(String(describing: firstObject))")
    }
    
    // Using NSDictionary
    func processDictionary() {
        let objcDict = NSDictionary(dictionary: ["key": "value"])
        let value = objcDict.object(forKey: "key")
        print("Value: \(String(describing: value))")
    }
    
    // Using NSNotificationCenter
    func setupNotifications() {
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleNotification(_:)),
            name: NSNotification.Name("CustomNotification"),
            object: nil
        )
    }
    
    @objc func handleNotification(_ notification: Notification) {
        print("Notification received: \(notification.name)")
    }
    
    // Using NSUserDefaults
    func savePreferences() {
        let defaults = UserDefaults.standard
        defaults.set("value", forKey: "key")
        defaults.synchronize()
    }
    
    // Using NSFileManager
    func checkFile() {
        let fileManager = FileManager.default
        let exists = fileManager.fileExists(atPath: "/path/to/file")
        print("File exists: \(exists)")
    }
}

// Using Objective-C protocol
@objc protocol CustomDelegate {
    func didComplete()
    @objc optional func didFail(error: Error)
}

// Bridging to Objective-C
@objc class BridgedClass: NSObject {
    
    @objc func objcAccessibleMethod() {
        print("Can be called from Objective-C")
    }
    
    @objc var objcAccessibleProperty: String = "visible to ObjC"
}

// Using Core Foundation (C API via Objective-C)
func useCoreFoundation() {
    let cfString = "Hello" as CFString
    let cfArray = [1, 2, 3] as CFArray
    print("CF String: \(cfString), CF Array: \(cfArray)")
}
