import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import RequestBuilder from '../RequestBuilder.svelte';
import * as api from '../api';

// Mock API calls
vi.mock('../api', () => ({
  getOpenAPISpec: vi.fn()
}));

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

const mockSpec = {
  paths: {
    '/users/{id}': {
      get: {
        parameters: [
          {
            name: 'id',
            in: 'path',
            required: true,
            schema: { type: 'integer' }
          },
          {
            name: 'fields',
            in: 'query',
            schema: { type: 'string' }
          },
          {
            name: 'Authorization',
            in: 'header',
            schema: { type: 'string' }
          }
        ],
        responses: {
          '200': {
            description: 'Success'
          }
        }
      },
      put: {
        parameters: [
          {
            name: 'id',
            in: 'path',
            required: true,
            schema: { type: 'integer' }
          }
        ],
        requestBody: {
          content: {
            'application/json': {
              schema: {
                type: 'object',
                properties: {
                  name: { type: 'string' },
                  email: { type: 'string' }
                }
              },
              example: {
                name: 'John Doe',
                email: 'john@example.com'
              }
            }
          }
        },
        responses: {
          '200': {
            description: 'Success'
          }
        }
      }
    }
  }
};

describe('RequestBuilder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getOpenAPISpec.mockImplementation(() => Promise.resolve(mockSpec));
    mockFetch.mockImplementation(() => Promise.resolve({
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: new Headers({ 'Content-Type': 'application/json' }),
      json: () => Promise.resolve({ success: true })
    }));

    // Mock timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders loading state initially', () => {
    render(RequestBuilder);
    expect(screen.getByText('Loading schemas...')).toBeInTheDocument();
  });

  it('loads and displays API endpoints', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
      expect(screen.getByText('GET')).toBeInTheDocument();
      expect(screen.getByText('PUT')).toBeInTheDocument();
    });
  });

  it('shows endpoint form when selected', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Check form fields
    await waitFor(() => {
      expect(screen.getByLabelText(/id/)).toBeInTheDocument();
      expect(screen.getByLabelText(/fields/)).toBeInTheDocument();
      expect(screen.getByLabelText(/Authorization/)).toBeInTheDocument();
    });
  });

  it('builds URL with path parameters', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill path parameter
    const idInput = screen.getByLabelText(/id/);
    fireEvent.input(idInput, { target: { value: '123' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Verify fetch call
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/users/123',
        expect.any(Object)
      );
    });
  });

  it('includes query parameters in URL', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill parameters
    const idInput = screen.getByLabelText(/id/);
    const fieldsInput = screen.getByLabelText(/fields/);
    fireEvent.input(idInput, { target: { value: '123' } });
    fireEvent.input(fieldsInput, { target: { value: 'name,email' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Verify fetch call
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/users/123?fields=name%2Cemail',
        expect.any(Object)
      );
    });
  });

  it('includes headers in request', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill parameters
    const idInput = screen.getByLabelText(/id/);
    const authInput = screen.getByLabelText(/Authorization/);
    fireEvent.input(idInput, { target: { value: '123' } });
    fireEvent.input(authInput, { target: { value: 'Bearer token' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Verify fetch call
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/users/123',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer token'
          })
        })
      );
    });
  });

  it('handles request body for PUT endpoints', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click PUT method
    const putButton = screen.getByText('PUT');
    fireEvent.click(putButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill path parameter
    const idInput = screen.getByLabelText(/id/);
    fireEvent.input(idInput, { target: { value: '123' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Load example request body
    const loadExampleButton = screen.getByText('Load Example');
    fireEvent.click(loadExampleButton);

    // Advance timers to trigger example load
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Verify fetch call
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/users/123',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({
            name: 'John Doe',
            email: 'john@example.com'
          })
        })
      );
    });
  });

  it('displays response after request', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill path parameter
    const idInput = screen.getByLabelText(/id/);
    fireEvent.input(idInput, { target: { value: '123' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Check response display
    await waitFor(() => {
      expect(screen.getByText('Response')).toBeInTheDocument();
      expect(screen.getByText('Status: 200 OK')).toBeInTheDocument();
      expect(screen.getByText(/success: true/)).toBeInTheDocument();
    });
  });

  it('handles request errors gracefully', async () => {
    // Mock fetch error
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill path parameter
    const idInput = screen.getByLabelText(/id/);
    fireEvent.input(idInput, { target: { value: '123' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Send request
    const sendButton = screen.getByText('Send Request');
    fireEvent.click(sendButton);

    // Advance timers to trigger request
    vi.advanceTimersByTime(0);

    // Check error display
    await waitFor(() => {
      expect(screen.getByText('Error: Network error')).toBeInTheDocument();
    });
  });

  it('resets form when endpoint changes', async () => {
    render(RequestBuilder);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger form update
    vi.advanceTimersByTime(0);

    // Fill parameters
    const idInput = screen.getByLabelText(/id/);
    fireEvent.input(idInput, { target: { value: '123' } });

    // Advance timers to trigger input update
    vi.advanceTimersByTime(0);

    // Switch to PUT method
    const putButton = screen.getByText('PUT');
    fireEvent.click(putButton);

    // Advance timers to trigger form reset
    vi.advanceTimersByTime(0);

    // Check if form was reset
    await waitFor(() => {
      const newIdInput = screen.getByLabelText(/id/);
      expect(newIdInput.value).toBe('');
    });
  });
}); 