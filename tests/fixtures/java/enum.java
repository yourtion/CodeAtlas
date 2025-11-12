package com.example.test;

/**
 * Enum representing days of the week.
 */
public enum DayOfWeek {
    /**
     * Monday
     */
    MONDAY,
    
    /**
     * Tuesday
     */
    TUESDAY,
    
    /**
     * Wednesday
     */
    WEDNESDAY,
    
    /**
     * Thursday
     */
    THURSDAY,
    
    /**
     * Friday
     */
    FRIDAY,
    
    /**
     * Saturday
     */
    SATURDAY,
    
    /**
     * Sunday
     */
    SUNDAY;
    
    /**
     * Checks if this is a weekend day.
     * @return true if weekend, false otherwise
     */
    public boolean isWeekend() {
        return this == SATURDAY || this == SUNDAY;
    }
}
