import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  fetchState,
  updateResource,
  deleteResource,
  createResource,
  fetchResource,
  fetchResourcesByType,
  getOpenAPISpec,
  getServerStatus,
  setHistoryCallback
} from '../api';

describe('API Service', () => {
  let mockFetch;
  let mockHistoryCallback;

  function createMockResponse(data, options = {}) {
    const response = {
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: new Headers({ 'Content-Type': 'application/json' }),
      ...options,
      clone: function() {
        return {
          ...this,
          json: () => Promise.resolve(data)
        };
      },
      json: () => Promise.resolve(data)
    };
    return Promise.resolve(response);
  }

  beforeEach(() => {
    mockFetch = vi.fn();
    vi.stubGlobal('fetch', mockFetch);
    mockHistoryCallback = null;
    setHistoryCallback(null);
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it('fetches state', async () => {
    const mockState = { users: [], posts: [] };
    mockFetch.mockImplementation(() => createMockResponse(mockState));

    const result = await fetchState();
    expect(result).toEqual(mockState);
    expect(mockFetch).toHaveBeenCalledWith('/api/state', expect.any(Object));
  });

  it('updates a resource', async () => {
    const mockResponse = { id: 1, name: 'Updated' };
    mockFetch.mockImplementation(() => createMockResponse(mockResponse));

    const type = 'users';
    const id = 1;
    const data = { name: 'Updated' };

    const result = await updateResource(type, id, data);
    expect(result).toEqual(mockResponse);
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/users/1',
      expect.objectContaining({
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
    );
  });

  it('deletes a resource', async () => {
    mockFetch.mockImplementation(() => createMockResponse(null));

    await deleteResource('users', 1);
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/users/1',
      expect.objectContaining({
        method: 'DELETE'
      })
    );
  });

  it('creates a resource', async () => {
    const mockResponse = { id: 1, name: 'New' };
    mockFetch.mockImplementation(() => createMockResponse(mockResponse));

    const type = 'users';
    const data = { name: 'New' };

    const result = await createResource(type, data);
    expect(result).toEqual(mockResponse);
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/users',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
    );
  });

  it('fetches a single resource', async () => {
    const mockResponse = { id: 1, name: 'User' };
    mockFetch.mockImplementation(() => createMockResponse(mockResponse));

    const result = await fetchResource('users', 1);
    expect(result).toEqual(mockResponse);
    expect(mockFetch).toHaveBeenCalledWith('/api/users/1', expect.any(Object));
  });

  it('fetches resources by type', async () => {
    const mockResponse = [
      { id: 1, name: 'User 1' },
      { id: 2, name: 'User 2' }
    ];
    mockFetch.mockImplementation(() => createMockResponse(mockResponse));

    const result = await fetchResourcesByType('users');
    expect(result).toEqual(mockResponse);
    expect(mockFetch).toHaveBeenCalledWith('/api/users', expect.any(Object));
  });

  it('fetches OpenAPI spec', async () => {
    const mockSpec = { paths: {} };
    mockFetch.mockImplementation(() => createMockResponse(mockSpec));

    const result = await getOpenAPISpec();
    expect(result).toEqual(mockSpec);
    expect(mockFetch).toHaveBeenCalledWith('/openapi.yaml', expect.any(Object));
  });

  it('fetches server status', async () => {
    const mockStatus = { status: 'online', version: '1.0.0' };
    mockFetch.mockImplementation(() => createMockResponse(mockStatus));

    const result = await getServerStatus();
    expect(result).toEqual(mockStatus);
    expect(mockFetch).toHaveBeenCalledWith('/api/status', expect.any(Object));
  });

  it('handles API errors', async () => {
    mockFetch.mockImplementation(() => createMockResponse(null, {
      ok: false,
      status: 404,
      statusText: 'Not Found'
    }));

    await expect(fetchState()).rejects.toThrow('Request failed: Not Found');
  });

  it('handles network errors', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    await expect(fetchState()).rejects.toThrow('Network error');
  });

  it('records request history when callback is set', async () => {
    const mockCallback = vi.fn();
    setHistoryCallback(mockCallback);

    const mockResponse = { id: 1, name: 'User' };
    mockFetch.mockImplementation(() => createMockResponse(mockResponse));

    await fetchResource('users', 1);

    expect(mockCallback).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'GET',
        url: '/api/users/1',
        headers: {}
      }),
      expect.objectContaining({
        status: 200,
        statusText: 'OK',
        headers: { 'Content-Type': 'application/json' },
        body: mockResponse
      })
    );
  });

  it('handles JSON parsing errors in history recording', async () => {
    const mockCallback = vi.fn();
    setHistoryCallback(mockCallback);

    mockFetch.mockImplementation(() => Promise.resolve({
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: new Headers({ 'Content-Type': 'application/json' }),
      clone: () => ({
        json: () => Promise.reject(new Error('Invalid JSON')),
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'Content-Type': 'application/json' })
      }),
      json: () => Promise.reject(new Error('Invalid JSON'))
    }));

    await expect(fetchResource('users', 1)).rejects.toThrow('Invalid JSON');

    expect(mockCallback).toHaveBeenCalledWith({
      method: 'GET',
      url: '/api/users/1',
      headers: {},
      body: undefined
    }, {
      status: 200,
      statusText: 'OK',
      headers: { 'Content-Type': 'application/json' },
      body: null
    });
  });
}); 