// Kotlin calling Java code
// This demonstrates Kotlin-Java interop in Android development

package com.example.interop

import java.util.ArrayList
import java.util.HashMap
import java.util.Date
import java.text.SimpleDateFormat
import java.io.File
import java.io.FileReader
import java.io.BufferedReader

// Using Java collections
class KotlinJavaInterop {
    
    // Using ArrayList (Java collection)
    fun useJavaArrayList() {
        val javaList = ArrayList<String>()
        javaList.add("item1")
        javaList.add("item2")
        val size = javaList.size
        val first = javaList.get(0)
        println("Size: $size, First: $first")
    }
    
    // Using HashMap (Java collection)
    fun useJavaHashMap() {
        val javaMap = HashMap<String, Int>()
        javaMap.put("key1", 100)
        javaMap.put("key2", 200)
        val value = javaMap.get("key1")
        println("Value: $value")
    }
    
    // Using Java Date
    fun useJavaDate() {
        val date = Date()
        val formatter = SimpleDateFormat("yyyy-MM-dd")
        val formatted = formatter.format(date)
        println("Date: $formatted")
    }
    
    // Using Java File I/O
    fun readJavaFile(path: String) {
        val file = File(path)
        val exists = file.exists()
        val reader = FileReader(file)
        val buffered = BufferedReader(reader)
        val line = buffered.readLine()
        buffered.close()
        println("Exists: $exists, Line: $line")
    }
    
    // Using Java String methods
    fun useJavaString() {
        val javaString = String("Hello")
        val length = javaString.length
        val upper = javaString.toUpperCase()
        val substring = javaString.substring(0, 3)
        println("Length: $length, Upper: $upper, Sub: $substring")
    }
    
    // Using Java System class
    fun useJavaSystem() {
        val currentTime = System.currentTimeMillis()
        val property = System.getProperty("user.home")
        System.out.println("Time: $currentTime, Home: $property")
    }
}

// Calling Java static methods
object JavaStaticCalls {
    fun callStaticMethods() {
        // Math class
        val max = Math.max(10, 20)
        val sqrt = Math.sqrt(16.0)
        
        // Integer class
        val parsed = Integer.parseInt("123")
        val hex = Integer.toHexString(255)
        
        // String class
        val formatted = String.format("Value: %d", 42)
        
        println("Max: $max, Sqrt: $sqrt, Parsed: $parsed, Hex: $hex, Formatted: $formatted")
    }
}

// Using Java interfaces
class JavaInterfaceImpl : Runnable {
    override fun run() {
        println("Running from Kotlin")
    }
    
    fun startThread() {
        val thread = Thread(this)
        thread.start()
    }
}

// Using Java exceptions
class JavaExceptionHandling {
    fun handleJavaException() {
        try {
            throw IllegalArgumentException("Java exception from Kotlin")
        } catch (e: IllegalArgumentException) {
            e.printStackTrace()
        }
    }
}

// Calling Java constructors
class JavaConstructorCalls {
    fun createJavaObjects() {
        val stringBuilder = StringBuilder()
        stringBuilder.append("Hello")
        stringBuilder.append(" World")
        
        val arrayList = ArrayList<Int>(10)
        val hashMap = HashMap<String, String>(16)
        
        println("StringBuilder: ${stringBuilder.toString()}")
    }
}
