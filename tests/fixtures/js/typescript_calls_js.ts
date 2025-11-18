// TypeScript calling JavaScript code
// This demonstrates TS-JS interop in frontend/Node.js projects


// Using JavaScript library without type definitions
// @ts-ignore
import oldLibrary from 'old-js-library';
import defaultExport from './default-export.js';
// Importing JavaScript modules
import * as jsModule from './legacy-module.js';
import { jsClass, jsFunction } from './utils.js';

// TypeScript class using JavaScript functions
class TypeScriptComponent {

  // Calling JavaScript function
  processData(data: any) {
    const result = jsFunction(data);
    console.log('Result:', result);
    return result;
  }

  // Using JavaScript class
  createJsInstance() {
    const instance = new jsClass();
    instance.doSomething();
    return instance;
  }

  // Using JavaScript module
  useJsModule() {
    const value = jsModule.getValue();
    jsModule.setValue(42);
    return value;
  }

  // Using default export from JS
  useDefaultExport() {
    const result = defaultExport();
    return result;
  }
}

// Using JavaScript global objects
function useJavaScriptGlobals() {
  // Console (JavaScript global)
  console.log('Logging from TypeScript');
  console.error('Error message');
  console.warn('Warning message');

  // setTimeout/setInterval (JavaScript timers)
  setTimeout(() => {
    console.log('Delayed execution');
  }, 1000);

  const intervalId = setInterval(() => {
    console.log('Repeated execution');
  }, 1000);
  clearInterval(intervalId);

  // Promise (JavaScript async)
  Promise.resolve(42)
    .then(value => console.log(value))
    .catch(error => console.error(error));
}

// Using JavaScript Array methods
function useJavaScriptArrays() {
  const arr: number[] = [1, 2, 3, 4, 5];

  // JavaScript array methods
  const mapped = arr.map(x => x * 2);
  const filtered = arr.filter(x => x > 2);
  const reduced = arr.reduce((acc, x) => acc + x, 0);
  const found = arr.find(x => x === 3);
  const some = arr.some(x => x > 3);
  const every = arr.every(x => x > 0);

  return { mapped, filtered, reduced, found, some, every };
}

// Using JavaScript String methods
function useJavaScriptStrings() {
  const str: string = "Hello World";

  // JavaScript string methods
  const upper = str.toUpperCase();
  const lower = str.toLowerCase();
  const split = str.split(' ');
  const substr = str.substring(0, 5);
  const includes = str.includes('World');
  const startsWith = str.startsWith('Hello');
  const endsWith = str.endsWith('World');
  const replaced = str.replace('World', 'TypeScript');

  return { upper, lower, split, substr, includes, startsWith, endsWith, replaced };
}

// Using JavaScript Object methods
function useJavaScriptObjects() {
  const obj = { a: 1, b: 2, c: 3 };

  // JavaScript object methods
  const keys = Object.keys(obj);
  const values = Object.values(obj);
  const entries = Object.entries(obj);
  const hasOwn = Object.hasOwnProperty.call(obj, 'a');
  const assigned = Object.assign({}, obj, { d: 4 });

  return { keys, values, entries, hasOwn, assigned };
}

// Using JavaScript JSON
function useJavaScriptJSON() {
  const data = { name: 'TypeScript', version: 5 };

  // JavaScript JSON methods
  const stringified = JSON.stringify(data);
  const parsed = JSON.parse(stringified);

  return { stringified, parsed };
}

// Using JavaScript Math
function useJavaScriptMath() {
  // JavaScript Math methods
  const max = Math.max(1, 2, 3);
  const min = Math.min(1, 2, 3);
  const random = Math.random();
  const floor = Math.floor(3.7);
  const ceil = Math.ceil(3.2);
  const round = Math.round(3.5);
  const sqrt = Math.sqrt(16);
  const pow = Math.pow(2, 3);

  return { max, min, random, floor, ceil, round, sqrt, pow };
}

// Using JavaScript Date
function useJavaScriptDate() {
  // JavaScript Date methods
  const now = new Date();
  const timestamp = Date.now();
  const year = now.getFullYear();
  const month = now.getMonth();
  const date = now.getDate();
  const time = now.getTime();
  const iso = now.toISOString();

  return { timestamp, year, month, date, time, iso };
}

// Using JavaScript RegExp
function useJavaScriptRegExp() {
  const pattern = /\d+/g;
  const text = "abc123def456";

  // JavaScript RegExp methods
  const test = pattern.test(text);
  const match = text.match(pattern);
  const replace = text.replace(pattern, 'X');
  const split = text.split(/\d+/);

  return { test, match, replace, split };
}

// Calling JavaScript with dynamic types
function callJavaScriptDynamic() {
  // Using 'any' to call JavaScript code
  const jsLib: any = oldLibrary;
  jsLib.init();
  jsLib.process({ data: 'value' });
  jsLib.cleanup();
}

// Using JavaScript fetch API
async function useJavaScriptFetch() {
  try {
    const response = await fetch('https://api.example.com/data');
    const json = await response.json();
    const text = await response.text();
    return { json, text };
  } catch (error) {
    console.error('Fetch error:', error);
  }
}

// Using JavaScript localStorage (browser)
function useJavaScriptLocalStorage() {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem('key', 'value');
    const value = localStorage.getItem('key');
    localStorage.removeItem('key');
    localStorage.clear();
    return value;
  }
}

// Using JavaScript require (Node.js)
function useJavaScriptRequire() {
  // CommonJS require
  const fs = require('fs');
  const path = require('path');
  const util = require('util');

  return { fs, path, util };
}

export {
  callJavaScriptDynamic, TypeScriptComponent, useJavaScriptArrays, useJavaScriptDate, useJavaScriptFetch, useJavaScriptGlobals, useJavaScriptJSON, useJavaScriptLocalStorage, useJavaScriptMath, useJavaScriptObjects, useJavaScriptRegExp, useJavaScriptRequire, useJavaScriptStrings
};

