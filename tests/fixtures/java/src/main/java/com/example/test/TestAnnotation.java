package com.example.test;

import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;

/**
 * Custom annotation for testing.
 */
@Retention(RetentionPolicy.RUNTIME)
public @interface TestAnnotation {
    String value() default "";
}

/**
 * Class using annotations.
 */
@TestAnnotation("test")
public class AnnotatedClass {
    
    @TestAnnotation("field")
    private String annotatedField;
    
    /**
     * Annotated method.
     */
    @TestAnnotation("method")
    public void annotatedMethod() {
        // Method implementation
    }
    
    @Override
    public String toString() {
        return "AnnotatedClass";
    }
}
