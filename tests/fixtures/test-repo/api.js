// API module with exports

const API_URL = 'https://api.example.com';

/**
 * Fetch data from API
 */
export async function fetchData() {
  const response = await fetch(API_URL);
  return response.json();
}

/**
 * Post data to API
 */
export async function postData(data) {
  const response = await fetch(API_URL, {
    method: 'POST',
    body: JSON.stringify(data)
  });
  return response.json();
}

/**
 * Helper class for API requests
 */
export class ApiClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  async get(endpoint) {
    return fetch(`${this.baseUrl}${endpoint}`);
  }

  async post(endpoint, data) {
    return fetch(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      body: JSON.stringify(data)
    });
  }
}
