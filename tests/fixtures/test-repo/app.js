// JavaScript test file with various constructs

import { fetchData } from './api.js';
import * as utils from './utils.js';

/**
 * Main application class
 */
class Application {
  constructor(name) {
    this.name = name;
    this.initialized = false;
  }

  /**
   * Initialize the application
   */
  async initialize() {
    console.log(`Initializing ${this.name}`);
    const data = await fetchData();
    this.initialized = true;
    return data;
  }

  /**
   * Get application status
   */
  getStatus() {
    return {
      name: this.name,
      initialized: this.initialized
    };
  }
}

/**
 * Process user input
 * @param {string} input - User input string
 * @returns {string} Processed output
 */
function processInput(input) {
  return input.trim().toLowerCase();
}

/**
 * Arrow function for data transformation
 */
const transformData = (data) => {
  return data.map(item => item.toUpperCase());
};

/**
 * Async arrow function
 */
const loadConfig = async () => {
  const config = await fetchData();
  return config;
};

export { Application, processInput, transformData };
export default Application;
