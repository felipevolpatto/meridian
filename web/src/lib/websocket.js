// websocket.js
let ws = null;
let reconnectTimeout = null;
const subscribers = new Set();
const messageQueue = [];
let isConnecting = false;

// For testing purposes
export function __resetWebSocket() {
  ws = null;
  reconnectTimeout = null;
  subscribers.clear();
  messageQueue.length = 0;
  isConnecting = false;
}

function connect() {
  if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
    return ws;
  }

  if (isConnecting) {
    return ws;
  }

  isConnecting = true;
  const protocol = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = typeof window !== 'undefined' ? window.location.host : 'localhost:3000';
  const url = `${protocol}//${host}/ws`;

  // Create WebSocket instance
  if (typeof WebSocket === 'undefined') {
    throw new Error('WebSocket is not supported in this environment');
  }

  ws = new WebSocket(url);

  // Set up event handlers
  const handleOpen = () => {
    isConnecting = false;
    notifySubscribers({ type: 'connection', status: 'connected' });
    processMessageQueue();
  };

  const handleClose = () => {
    isConnecting = false;
    notifySubscribers({ type: 'connection', status: 'disconnected' });
    if (!reconnectTimeout) {
      reconnectTimeout = setTimeout(() => {
        reconnectTimeout = null;
        connect();
      }, 5000);
    }
  };

  const handleError = () => {
    isConnecting = false;
    notifySubscribers({
      type: 'connection',
      status: 'error',
      error: 'Connection failed'
    });
  };

  const handleMessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      notifySubscribers(message);
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  };

  ws.onopen = handleOpen;
  ws.onclose = handleClose;
  ws.onerror = handleError;
  ws.onmessage = handleMessage;

  return ws;
}

function notifySubscribers(message) {
  subscribers.forEach(callback => callback(message));
}

function processMessageQueue() {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    return;
  }

  // Process all queued messages
  const messages = [...messageQueue];
  messageQueue.length = 0;

  for (const message of messages) {
    try {
      ws.send(JSON.stringify(message));
    } catch (error) {
      console.error('Failed to send message:', error);
      // Put the remaining messages back in the queue
      messageQueue.push(message, ...messages.slice(messages.indexOf(message) + 1));
      break;
    }
  }
}

export function subscribe(callback) {
  subscribers.add(callback);
  if (!ws || ws.readyState === WebSocket.CLOSED) {
    const newWs = connect();
    if (newWs) {
      ws = newWs;
    }
  }
  return () => {
    subscribers.delete(callback);
    if (subscribers.size === 0 && ws) {
      ws.close();
      ws = null;
    }
  };
}

export function send(message) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    messageQueue.push(message);
    if (!ws || ws.readyState === WebSocket.CLOSED) {
      ws = connect();
    }
    return;
  }
  ws.send(JSON.stringify(message));
}

export function getConnectionStatus() {
  if (!ws) return 'disconnected';
  if (typeof WebSocket === 'undefined') return 'unknown';
  
  // Handle mock WebSocket in tests
  if (process.env.NODE_ENV === 'test') {
    return ws.readyState === WebSocket.OPEN ? 'connected' :
           ws.readyState === WebSocket.CONNECTING ? 'connecting' :
           'disconnected';
  }
  
  switch (ws.readyState) {
    case WebSocket.CONNECTING:
      return 'connecting';
    case WebSocket.OPEN:
      return 'connected';
    case WebSocket.CLOSING:
    case WebSocket.CLOSED:
      return 'disconnected';
    default:
      return 'unknown';
  }
}

// Clean up on module unload
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }
    if (ws) {
      ws.close();
    }
  });
} 