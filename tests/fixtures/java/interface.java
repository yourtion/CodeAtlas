package com.example.test;

/**
 * A simple interface for testing.
 */
public interface Drawable {
    /**
     * Draws the object.
     */
    void draw();
    
    /**
     * Gets the color.
     * @return The color as a string
     */
    String getColor();
    
    /**
     * Sets the position.
     * @param x The x coordinate
     * @param y The y coordinate
     */
    void setPosition(int x, int y);
}

/**
 * Extended interface for advanced drawing.
 */
public interface AdvancedDrawable extends Drawable {
    /**
     * Draws with effects.
     * @param effect The effect to apply
     */
    void drawWithEffect(String effect);
}
