// api.js
const API_BASE = '/api';

let historyCallback = null;

export function setHistoryCallback(callback) {
  historyCallback = callback;
}

async function makeRequest(url, options = {}) {
  const request = {
    method: options.method || 'GET',
    url,
    headers: options.headers || {},
    body: options.body
  };

  try {
    const response = await fetch(url, options);
    
    if (!response.ok) {
      throw new Error(`Request failed: ${response.statusText}`);
    }

    // Try to parse the response for history recording
    const clonedResponse = response.clone();
    let responseBody = null;
    try {
      responseBody = await clonedResponse.json();
    } catch (error) {
      // If JSON parsing fails, set body to null
      responseBody = null;
    }

    // Record history before potentially failing on JSON parse
    if (historyCallback) {
      const historyResponse = {
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries()),
        body: responseBody
      };
      historyCallback(request, historyResponse);
    }

    // Parse the actual response
    return await response.json();
  } catch (error) {
    if (error.message.includes('Request failed')) {
      throw error;
    }
    throw new Error(error.message);
  }
}

/**
 * Fetches the current state of all resources
 * @returns {Promise<Object>} The current state
 */
export async function fetchState() {
  return makeRequest('/api/state');
}

/**
 * Updates a resource
 * @param {string} type - The resource type
 * @param {string|number} id - The resource ID
 * @param {Object} data - The updated resource data
 * @returns {Promise<Object>} The updated resource
 */
export async function updateResource(type, id, data) {
  return makeRequest(`/api/${type}/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  });
}

/**
 * Deletes a resource
 * @param {string} type - The resource type
 * @param {string|number} id - The resource ID
 * @returns {Promise<void>}
 */
export async function deleteResource(type, id) {
  return makeRequest(`/api/${type}/${id}`, {
    method: 'DELETE'
  });
}

/**
 * Creates a new resource
 * @param {string} type - The resource type
 * @param {Object} data - The resource data
 * @returns {Promise<Object>} The created resource
 */
export async function createResource(type, data) {
  return makeRequest(`/api/${type}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  });
}

/**
 * Fetches a single resource
 * @param {string} type - The resource type
 * @param {string|number} id - The resource ID
 * @returns {Promise<Object>} The resource
 */
export async function fetchResource(type, id) {
  return makeRequest(`/api/${type}/${id}`);
}

/**
 * Fetches all resources of a type
 * @param {string} type - The resource type
 * @returns {Promise<Array>} The resources
 */
export async function fetchResourcesByType(type) {
  return makeRequest(`/api/${type}`);
}

/**
 * Fetches the OpenAPI specification
 * @returns {Promise<Object>} The OpenAPI spec
 */
export async function getOpenAPISpec() {
  return makeRequest('/openapi.yaml');
}

/**
 * Fetches the server status
 * @returns {Promise<Object>} The server status
 */
export async function getServerStatus() {
  return makeRequest('/api/status');
} 