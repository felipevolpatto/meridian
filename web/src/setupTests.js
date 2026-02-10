import '@testing-library/jest-dom';
import { vi, afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';

// Mock fetch
global.fetch = vi.fn();

// Mock Headers
class MockHeaders {
  constructor(init = {}) {
    this.headers = new Map(Object.entries(init));
  }

  append(name, value) {
    this.headers.set(name, value);
  }

  delete(name) {
    this.headers.delete(name);
  }

  get(name) {
    return this.headers.get(name);
  }

  has(name) {
    return this.headers.has(name);
  }

  set(name, value) {
    this.headers.set(name, value);
  }

  entries() {
    return this.headers.entries();
  }

  keys() {
    return this.headers.keys();
  }

  values() {
    return this.headers.values();
  }

  forEach(callback) {
    this.headers.forEach(callback);
  }
}

global.Headers = MockHeaders;

// Mock WebSocket
class MockWebSocket {
  constructor(url) {
    this.url = url;
    this.readyState = 0; // CONNECTING
    this.send = vi.fn();
    this.close = vi.fn();
    this.onopen = null;
    this.onclose = null;
    this.onmessage = null;
    this.onerror = null;
    this.binaryType = 'blob';
    this.bufferedAmount = 0;
    this.extensions = '';
    this.protocol = '';
  }

  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
}

// Replace global WebSocket
global.WebSocket = MockWebSocket;

// Mock localStorage
class MockStorage {
  constructor() {
    this.store = new Map();
  }

  get length() {
    return this.store.size;
  }

  key(n) {
    return Array.from(this.store.keys())[n];
  }

  getItem(key) {
    return this.store.get(key) || null;
  }

  setItem(key, value) {
    this.store.set(key, value);
  }

  removeItem(key) {
    this.store.delete(key);
  }

  clear() {
    this.store.clear();
  }
}

const mockStorage = new MockStorage();

Object.defineProperty(global, 'localStorage', {
  value: mockStorage,
  writable: true
});

// Clean up after each test
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mockStorage.clear();
});

// Mock timers
vi.useFakeTimers();

// Export mocks for tests
export {
  MockWebSocket,
  mockStorage as localStorage,
  MockHeaders
}; 