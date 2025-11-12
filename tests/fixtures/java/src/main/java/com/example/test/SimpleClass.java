package com.example.test;

import java.util.List;
import java.util.ArrayList;

/**
 * A simple class for testing Java parser.
 * This class demonstrates basic Java features.
 */
public class SimpleClass {
    private String name;
    private int age;
    
    /**
     * Constructor for SimpleClass.
     * @param name The name of the person
     * @param age The age of the person
     */
    public SimpleClass(String name, int age) {
        this.name = name;
        this.age = age;
    }
    
    /**
     * Gets the name.
     * @return The name
     */
    public String getName() {
        return name;
    }
    
    /**
     * Sets the name.
     * @param name The new name
     */
    public void setName(String name) {
        this.name = name;
    }
    
    /**
     * Calculates something.
     */
    public int calculate() {
        int result = age * 2;
        return result;
    }
    
    /**
     * Processes a list of items.
     */
    public void processList() {
        List<String> items = new ArrayList<>();
        items.add(name);
    }
}
