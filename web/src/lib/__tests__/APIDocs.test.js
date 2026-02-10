import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import APIDocs from '../APIDocs.svelte';
import * as api from '../api';

describe('APIDocs', () => {
  const mockSpec = {
    paths: {
      '/api/users': {
        get: {
          summary: 'List users',
          description: 'Get a list of all users',
          parameters: [
            {
              name: 'limit',
              in: 'query',
              description: 'Maximum number of items to return',
              schema: { type: 'integer' }
            }
          ],
          responses: {
            '200': {
              description: 'Successful response',
              content: {
                'application/json': {
                  schema: { $ref: '#/components/schemas/UserList' }
                }
              }
            }
          }
        },
        post: {
          summary: 'Create user',
          description: 'Create a new user',
          requestBody: {
            content: {
              'application/json': {
                schema: { $ref: '#/components/schemas/User' }
              }
            }
          },
          responses: {
            '201': {
              description: 'User created',
              content: {
                'application/json': {
                  schema: { $ref: '#/components/schemas/User' }
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
            name: { type: 'string' }
          }
        },
        UserList: {
          type: 'array',
          items: { $ref: '#/components/schemas/User' }
        }
      }
    }
  };

  beforeEach(() => {
    vi.spyOn(api, 'getOpenAPISpec').mockImplementation(() => {
      return Promise.resolve(mockSpec);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders loading state initially', () => {
    render(APIDocs);
    expect(screen.getByText('Loading API documentation...')).toBeInTheDocument();
  });

  it('loads and displays API specification', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
      expect(screen.getByText('GET')).toBeInTheDocument();
      expect(screen.getByText('POST')).toBeInTheDocument();
    });
  });

  it('shows endpoint details when selected', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });

    const getEndpoint = screen.getByRole('button', { name: /GET \/api\/users/i });
    await fireEvent.click(getEndpoint);

    await waitFor(() => {
      expect(screen.getByText('List users')).toBeInTheDocument();
      expect(screen.getByText('Get a list of all users')).toBeInTheDocument();
    });
  });

  it('displays parameters for selected endpoint', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });

    const getEndpoint = screen.getByRole('button', { name: /GET \/api\/users/i });
    await fireEvent.click(getEndpoint);

    await waitFor(() => {
      expect(screen.getByText('limit')).toBeInTheDocument();
      expect(screen.getByText('Maximum number of items to return')).toBeInTheDocument();
      expect(screen.getByText('query')).toBeInTheDocument();
      expect(screen.getByText('integer')).toBeInTheDocument();
    });
  });

  it('displays request body for POST endpoints', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });

    const postEndpoint = screen.getByRole('button', { name: /POST \/api\/users/i });
    await fireEvent.click(postEndpoint);

    await waitFor(() => {
      expect(screen.getByText('Request Body')).toBeInTheDocument();
      expect(screen.getByText('application/json')).toBeInTheDocument();
      expect(screen.getByText('#/components/schemas/User')).toBeInTheDocument();
    });
  });

  it('displays response information', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });

    const getEndpoint = screen.getByRole('button', { name: /GET \/api\/users/i });
    await fireEvent.click(getEndpoint);

    await waitFor(() => {
      expect(screen.getByText('Responses')).toBeInTheDocument();
      expect(screen.getByText('200')).toBeInTheDocument();
      expect(screen.getByText('Successful response')).toBeInTheDocument();
      expect(screen.getByText('#/components/schemas/UserList')).toBeInTheDocument();
    });
  });

  it('shows schemas sidebar', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('Schemas')).toBeInTheDocument();
      expect(screen.getByText('User')).toBeInTheDocument();
      expect(screen.getByText('UserList')).toBeInTheDocument();
    });
  });

  it('expands schema details when clicked', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('User')).toBeInTheDocument();
    });

    const userSchema = screen.getByRole('button', { name: /User/i });
    await fireEvent.click(userSchema);

    await waitFor(() => {
      expect(screen.getByText('Properties')).toBeInTheDocument();
      expect(screen.getByText('id')).toBeInTheDocument();
      expect(screen.getByText('name')).toBeInTheDocument();
    });
  });

  it('filters endpoints by search term', async () => {
    render(APIDocs);
    
    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText('Search endpoints...');
    await fireEvent.input(searchInput, { target: { value: 'user' } });

    await waitFor(() => {
      expect(screen.getByText('/api/users')).toBeInTheDocument();
    });
  });

  it('handles API errors gracefully', async () => {
    vi.spyOn(api, 'getOpenAPISpec').mockRejectedValue(new Error('Failed to load spec'));
    render(APIDocs);

    await waitFor(() => {
      expect(screen.getByText('Error loading API documentation')).toBeInTheDocument();
    });
  });
}); 