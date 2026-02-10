import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import MockData from '../MockData.svelte';
import * as api from '../api';

// Mock API calls
vi.mock('../api', () => ({
  getOpenAPISpec: vi.fn()
}));

// Mock clipboard API
const mockClipboard = {
  writeText: vi.fn()
};
Object.defineProperty(navigator, 'clipboard', {
  value: mockClipboard,
  writable: true
});

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
          }
        ],
        responses: {
          '200': {
            description: 'Success',
            content: {
              'application/json': {
                schema: {
                  type: 'object',
                  properties: {
                    id: { type: 'integer', example: 123 },
                    name: { type: 'string', example: 'John Doe' },
                    email: { type: 'string', example: 'john@example.com' }
                  }
                }
              }
            }
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
              }
            }
          }
        },
        responses: {
          '200': {
            description: 'Success',
            content: {
              'application/json': {
                schema: {
                  type: 'object',
                  properties: {
                    success: { type: 'boolean', example: true }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  components: {
    schemas: {
      User: {
        type: 'object',
        properties: {
          id: { type: 'integer' },
          name: { type: 'string' },
          email: { type: 'string' }
        }
      }
    }
  }
};

describe('MockData', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getOpenAPISpec.mockImplementation(() => {
      return new Promise((resolve) => {
        setTimeout(() => resolve(mockSpec), 0);
      });
    });
    mockClipboard.writeText.mockImplementation(() => {
      return new Promise((resolve) => {
        setTimeout(() => resolve(), 0);
      });
    });

    // Mock timers
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders loading state initially', () => {
    render(MockData);
    expect(screen.getByText('Loading schemas...')).toBeInTheDocument();
  });

  it('loads and displays API endpoints', async () => {
    render(MockData);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
      expect(screen.getByText('GET')).toBeInTheDocument();
      expect(screen.getByText('PUT')).toBeInTheDocument();
    });
  });

  it('generates mock data when endpoint is selected', async () => {
    render(MockData);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger mock data generation
    vi.advanceTimersByTime(0);

    // Check mock data
    await waitFor(() => {
      const mockJson = screen.getByText(/{\s+"body":\s+{\s+"id":\s+123,\s+"name":\s+"John Doe",\s+"email":\s+"john@example.com"\s+}\s+}/);
      expect(mockJson).toBeInTheDocument();
    });
  });

  it('copies mock data to clipboard', async () => {
    render(MockData);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger mock data generation
    vi.advanceTimersByTime(0);

    // Click copy button
    const copyButton = screen.getByText('Copy to Clipboard');
    fireEvent.click(copyButton);

    // Verify clipboard API call
    await waitFor(() => {
      expect(mockClipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('John Doe'));
    });
  });

  it('regenerates mock data on button click', async () => {
    render(MockData);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    await waitFor(() => {
      expect(screen.getByText('/users/{id}')).toBeInTheDocument();
    });

    // Click GET method
    const getButton = screen.getByText('GET');
    fireEvent.click(getButton);

    // Advance timers to trigger mock data generation
    vi.advanceTimersByTime(0);

    // Get initial mock data
    const initialMockData = screen.getByText(/{\s+"body":\s+{\s+"id":\s+123,\s+"name":\s+"John Doe",\s+"email":\s+"john@example.com"\s+}\s+}/);
    expect(initialMockData).toBeInTheDocument();

    // Click regenerate button
    const regenerateButton = screen.getByText('Regenerate');
    fireEvent.click(regenerateButton);

    // Advance timers to trigger mock data regeneration
    vi.advanceTimersByTime(0);

    // Check if mock data was regenerated
    await waitFor(() => {
      const newMockData = screen.getByText(/{\s+"body":\s+{\s+"id":\s+123,\s+"name":\s+"John Doe",\s+"email":\s+"john@example.com"\s+}\s+}/);
      expect(newMockData).toBeInTheDocument();
    });
  });

  it('handles API errors gracefully', async () => {
    // Mock API error
    api.getOpenAPISpec.mockRejectedValueOnce(new Error('Failed to load spec'));

    render(MockData);

    // Advance timers to trigger API call
    vi.advanceTimersByTime(0);

    // Check error message
    await waitFor(() => {
      expect(screen.getByText('Error: Failed to load spec')).toBeInTheDocument();
    });
  });
});