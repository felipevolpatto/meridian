import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
let subscribe, send, getConnectionStatus;

describe('WebSocket Service', () => {
  let mockWebSocket;

  beforeEach(() => {
    // Reset mocks
    mockWebSocket = {
      url: '',
      readyState: 0, // CONNECTING
      send: vi.fn(),
      close: vi.fn(),
      onopen: null,
      onclose: null,
      onmessage: null,
      onerror: null,
      // Helper methods for tests
      triggerOpen() {
        this.readyState = 1; // OPEN
        if (this.onopen) {
          this.onopen({ target: this });
        }
      },
      triggerClose() {
        this.readyState = 3; // CLOSED
        if (this.onclose) {
          this.onclose({ target: this });
        }
      },
      triggerMessage(data) {
        if (this.onmessage) {
          this.onmessage({ target: this, data });
        }
      },
      triggerError() {
        if (this.onerror) {
          this.onerror({ target: this });
        }
      }
    };

    // Mock WebSocket constructor and constants
    const WebSocketMock = vi.fn(() => mockWebSocket);
    WebSocketMock.CONNECTING = 0;
    WebSocketMock.OPEN = 1;
    WebSocketMock.CLOSING = 2;
    WebSocketMock.CLOSED = 3;
    vi.stubGlobal('WebSocket', WebSocketMock);

    // Mock process.env.NODE_ENV
    vi.stubGlobal('process', { env: { NODE_ENV: 'test' } });

    // Reset WebSocket instance
    vi.resetModules();
    vi.clearAllMocks();

    // Import WebSocket module after resetting
    const websocketModule = require('../websocket');
    subscribe = websocketModule.subscribe;
    send = websocketModule.send;
    getConnectionStatus = websocketModule.getConnectionStatus;

    // Reset WebSocket instance
    websocketModule.__resetWebSocket();

    // Mock window.location
    vi.stubGlobal('window', {
      location: {
        protocol: 'http:',
        host: 'localhost:3000'
      }
    });

    // Mock timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  it('establishes WebSocket connection', () => {
    subscribe(() => {});
    expect(WebSocket).toHaveBeenCalledWith('ws://localhost:3000/ws');
  });

  it('notifies subscribers of connection status', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate connection
    mockWebSocket.triggerOpen();
    expect(callback).toHaveBeenCalledWith({ type: 'connection', status: 'connected' });

    // Simulate disconnection
    mockWebSocket.triggerClose();
    expect(callback).toHaveBeenCalledWith({ type: 'connection', status: 'disconnected' });

    unsubscribe();
  });

  it('notifies subscribers of messages', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate connection
    mockWebSocket.triggerOpen();

    // Simulate message
    const message = { type: 'update', data: { id: 1 } };
    mockWebSocket.triggerMessage(JSON.stringify(message));

    expect(callback).toHaveBeenCalledWith(message);

    unsubscribe();
  });

  it('handles message parsing errors', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate connection
    mockWebSocket.triggerOpen();

    // Clear callback history
    callback.mockClear();

    // Simulate invalid message
    mockWebSocket.triggerMessage('invalid json');

    expect(callback).not.toHaveBeenCalled();

    unsubscribe();
  });

  it('queues messages when connection is not open', () => {
    const message = { type: 'test' };
    
    // Subscribe to establish connection
    const unsubscribe = subscribe(() => {});

    // Wait for initial connection
    vi.advanceTimersByTime(0);
    
    // Send message before connection
    send(message);
    expect(mockWebSocket.send).not.toHaveBeenCalled();

    // Simulate connection
    mockWebSocket.triggerOpen();
    expect(mockWebSocket.send).toHaveBeenCalledWith(JSON.stringify(message));

    unsubscribe();
  });

  it('sends messages immediately when connected', () => {
    const message = { type: 'test' };
    
    // Subscribe and simulate connection
    const unsubscribe = subscribe(() => {});

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate connection
    mockWebSocket.triggerOpen();

    // Send message
    send(message);
    expect(mockWebSocket.send).toHaveBeenCalledWith(JSON.stringify(message));

    unsubscribe();
  });

  it('attempts to reconnect on connection close', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate initial connection
    mockWebSocket.triggerOpen();
    expect(callback).toHaveBeenCalledWith({ type: 'connection', status: 'connected' });

    // Simulate disconnection
    mockWebSocket.triggerClose();
    expect(callback).toHaveBeenCalledWith({ type: 'connection', status: 'disconnected' });

    // Wait for reconnection attempt
    vi.advanceTimersByTime(5000);
    expect(WebSocket).toHaveBeenCalledTimes(2);

    unsubscribe();
  });

  it('notifies subscribers of connection errors', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Simulate connection error
    mockWebSocket.triggerError();
    expect(callback).toHaveBeenCalledWith({
      type: 'connection',
      status: 'error',
      error: 'Connection failed'
    });

    unsubscribe();
  });

  it('returns correct connection status', () => {
    // Subscribe to establish connection
    const unsubscribe = subscribe(() => {});

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Test different connection states
    mockWebSocket.readyState = WebSocket.OPEN;
    expect(getConnectionStatus()).toBe('connected');

    mockWebSocket.readyState = WebSocket.CLOSED;
    expect(getConnectionStatus()).toBe('disconnected');

    mockWebSocket.readyState = WebSocket.CONNECTING;
    expect(getConnectionStatus()).toBe('connecting');

    unsubscribe();
  });

  it('removes subscribers on unsubscribe', () => {
    const callback = vi.fn();
    const unsubscribe = subscribe(callback);

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    unsubscribe();

    mockWebSocket.triggerMessage(JSON.stringify({ type: 'test' }));
    expect(callback).not.toHaveBeenCalled();
  });

  it('processes message queue on connection', () => {
    const message1 = { type: 'test1' };
    const message2 = { type: 'test2' };

    // Subscribe to establish connection
    const unsubscribe = subscribe(() => {});

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Queue messages before connection
    send(message1);
    send(message2);
    expect(mockWebSocket.send).not.toHaveBeenCalled();

    // Simulate connection
    mockWebSocket.triggerOpen();

    // Verify messages were sent in order
    expect(mockWebSocket.send).toHaveBeenCalledTimes(2);
    expect(mockWebSocket.send).toHaveBeenNthCalledWith(1, JSON.stringify(message1));
    expect(mockWebSocket.send).toHaveBeenNthCalledWith(2, JSON.stringify(message2));

    unsubscribe();
  });

  it('prevents duplicate connections', () => {
    // Create two subscriptions
    const unsubscribe1 = subscribe(() => {});
    const unsubscribe2 = subscribe(() => {});

    // Wait for initial connection
    vi.advanceTimersByTime(0);

    // Verify only one WebSocket was created
    expect(WebSocket).toHaveBeenCalledTimes(1);

    // Clean up
    unsubscribe1();
    unsubscribe2();
  });
}); 