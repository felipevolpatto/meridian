import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import StateInspector from '../StateInspector.svelte';
import * as api from '../api';
import * as websocket from '../websocket';

// Mock API calls
vi.mock('../api', () => ({
  fetchState: vi.fn(),
  deleteResource: vi.fn(),
  updateResource: vi.fn(),
}));

// Mock WebSocket service
vi.mock('../websocket', () => ({
  subscribe: vi.fn(),
  getConnectionStatus: vi.fn(),
}));

const mockState = {
  users: [
    { id: 1, name: 'John Doe', email: 'john@example.com' },
    { id: 2, name: 'Jane Smith', email: 'jane@example.com' },
  ],
  posts: [
    { id: 1, title: 'First Post', content: 'Hello World' },
    { id: 2, title: 'Second Post', content: 'Another post' },
  ],
};

describe('StateInspector', () => {
  let unsubscribe;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    
    // Setup default mock implementations
    api.fetchState.mockImplementation(() => Promise.resolve(mockState));
    unsubscribe = vi.fn();
    websocket.subscribe.mockReturnValue(unsubscribe);
    websocket.getConnectionStatus.mockReturnValue('connected');

    // Mock timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  it('renders initial state correctly', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
      expect(screen.getByText('posts (2)')).toBeInTheDocument();
    });

    // Check connection status
    expect(screen.getByText('connected')).toBeInTheDocument();
  });

  it('filters resources by search term', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Enter search term
    const searchInput = screen.getByPlaceholderText('Search resources...');
    fireEvent.input(searchInput, { target: { value: 'John' } });

    // Advance timers to trigger filter
    vi.advanceTimersByTime(0);

    // Check filtered results
    await waitFor(() => {
      expect(screen.getByText('ID: 1')).toBeInTheDocument();
      expect(screen.queryByText('ID: 2')).not.toBeInTheDocument();
    });
  });

  it('filters resources by type', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Select users type
    const typeSelect = screen.getByRole('combobox');
    fireEvent.change(typeSelect, { target: { value: 'users' } });

    // Advance timers to trigger filter
    vi.advanceTimersByTime(0);

    // Check filtered results
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
      expect(screen.queryByText('posts')).not.toBeInTheDocument();
    });
  });

  it('sorts resources in ascending and descending order', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Change sort order to descending
    const sortSelect = screen.getAllByRole('combobox')[1];
    fireEvent.change(sortSelect, { target: { value: 'desc' } });

    // Advance timers to trigger sort
    vi.advanceTimersByTime(0);

    // Verify sort order (implementation depends on your sorting logic)
    const items = screen.getAllByText(/ID: \d/);
    expect(items[0].textContent).toContain('ID: 2');
    expect(items[1].textContent).toContain('ID: 1');
  });

  it('expands and collapses resource details', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Find and click expand button
    const expandButtons = screen.getAllByText('▶');
    fireEvent.click(expandButtons[0]);

    // Advance timers to trigger expand
    vi.advanceTimersByTime(0);

    // Check if details are shown
    await waitFor(() => {
      expect(screen.getByText('name:')).toBeInTheDocument();
      expect(screen.getByText('email:')).toBeInTheDocument();
    });

    // Collapse details
    const collapseButton = screen.getByText('▼');
    fireEvent.click(collapseButton);

    // Advance timers to trigger collapse
    vi.advanceTimersByTime(0);

    // Check if details are hidden
    await waitFor(() => {
      expect(screen.queryByText('name:')).not.toBeInTheDocument();
    });
  });

  it('handles resource editing', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Select a resource
    const resource = screen.getAllByText(/ID: \d/)[0];
    fireEvent.click(resource);

    // Advance timers to trigger selection
    vi.advanceTimersByTime(0);

    // Click edit button
    const editButton = screen.getByText('Edit');
    fireEvent.click(editButton);

    // Advance timers to trigger edit mode
    vi.advanceTimersByTime(0);

    // Edit the resource
    const textarea = screen.getByRole('textbox');
    const updatedData = {
      id: 1,
      name: 'Updated Name',
      email: 'updated@example.com'
    };
    fireEvent.input(textarea, { target: { value: JSON.stringify(updatedData, null, 2) } });

    // Save changes
    const saveButton = screen.getByText('Save');
    fireEvent.click(saveButton);

    // Advance timers to trigger save
    vi.advanceTimersByTime(0);

    // Verify API call
    await waitFor(() => {
      expect(api.updateResource).toHaveBeenCalledWith('users', 1, updatedData);
    });
  });

  it('handles resource deletion', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Select a resource
    const resource = screen.getAllByText(/ID: \d/)[0];
    fireEvent.click(resource);

    // Advance timers to trigger selection
    vi.advanceTimersByTime(0);

    // Delete the resource
    const deleteButton = screen.getByText('Delete');
    fireEvent.click(deleteButton);

    // Advance timers to trigger deletion
    vi.advanceTimersByTime(0);

    // Verify API call
    await waitFor(() => {
      expect(api.deleteResource).toHaveBeenCalledWith('users', 1);
    });
  });

  it('handles WebSocket updates', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Get the WebSocket callback
    const [callback] = websocket.subscribe.mock.calls[0];

    // Simulate state update message
    callback({ type: 'state_update' });

    // Advance timers to trigger state refresh
    vi.advanceTimersByTime(0);

    // Verify state refresh
    await waitFor(() => {
      expect(api.fetchState).toHaveBeenCalledTimes(2); // Initial + update
    });
  });

  it('handles WebSocket connection status changes', async () => {
    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Wait for initial state to load
    await waitFor(() => {
      expect(screen.getByText('users (2)')).toBeInTheDocument();
    });

    // Get the WebSocket callback
    const [callback] = websocket.subscribe.mock.calls[0];

    // Simulate disconnection
    callback({ type: 'connection', status: 'disconnected' });

    // Advance timers to trigger status update
    vi.advanceTimersByTime(0);

    // Check status indicator
    await waitFor(() => {
      expect(screen.getByText('disconnected')).toBeInTheDocument();
    });

    // Simulate reconnection
    callback({ type: 'connection', status: 'connected' });

    // Advance timers to trigger status update and state refresh
    vi.advanceTimersByTime(0);

    // Check status indicator and state refresh
    await waitFor(() => {
      expect(screen.getByText('connected')).toBeInTheDocument();
      expect(api.fetchState).toHaveBeenCalledTimes(2); // Initial + reconnect
    });
  });

  it('handles API errors gracefully', async () => {
    // Mock API error
    api.fetchState.mockRejectedValueOnce(new Error('Failed to fetch state'));

    render(StateInspector);

    // Advance timers to trigger state load
    vi.advanceTimersByTime(0);

    // Check error message
    await waitFor(() => {
      expect(screen.getByText('Failed to fetch state')).toBeInTheDocument();
    });
  });

  it('cleans up WebSocket subscription on unmount', async () => {
    const { unmount } = render(StateInspector);

    // Advance timers to trigger initial setup
    vi.advanceTimersByTime(0);

    // Wait for initial setup
    await waitFor(() => {
      expect(websocket.subscribe).toHaveBeenCalled();
    });

    // Unmount component
    unmount();

    // Advance timers to trigger cleanup
    vi.advanceTimersByTime(0);

    // Verify cleanup
    expect(unsubscribe).toHaveBeenCalled();
  });
}); 