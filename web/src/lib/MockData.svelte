<!-- MockData.svelte -->
<script>
  import { onMount } from 'svelte';
  import { getOpenAPISpec } from './api';

  let spec = null;
  let error = null;
  let selectedPath = null;
  let selectedMethod = null;
  let mockData = '';

  onMount(async () => {
    try {
      spec = await getOpenAPISpec();
    } catch (err) {
      error = err.message;
    }
  });

  function getMethodColor(method) {
    const colors = {
      get: '#61affe',
      post: '#49cc90',
      put: '#fca130',
      delete: '#f93e3e',
      patch: '#50e3c2',
      options: '#0d5aa7',
      head: '#9012fe',
      trace: '#9012fe'
    };
    return colors[method.toLowerCase()] || '#999';
  }

  function selectEndpoint(path, method) {
    selectedPath = path;
    selectedMethod = method;
    generateMockData();
  }

  function generateMockData() {
    if (!selectedPath || !selectedMethod) return;

    const endpoint = spec.paths[selectedPath][selectedMethod];
    const mockResponse = {};

    // Get success response schema
    const successResponse = Object.entries(endpoint.responses)
      .find(([code]) => code.startsWith('2'))?.[1];

    if (successResponse?.content?.['application/json']?.schema) {
      const schema = successResponse.content['application/json'].schema;
      mockResponse.body = generateMockFromSchema(schema);
    }

    mockData = JSON.stringify(mockResponse, null, 2);
  }

  function generateMockFromSchema(schema) {
    if (schema.$ref) {
      const refName = schema.$ref.split('/').pop();
      const refSchema = spec.components?.schemas?.[refName];
      return generateMockFromSchema(refSchema);
    }

    switch (schema.type) {
      case 'object':
        const obj = {};
        if (schema.properties) {
          Object.entries(schema.properties).forEach(([key, prop]) => {
            obj[key] = generateMockFromSchema(prop);
          });
        }
        return obj;

      case 'array':
        return [generateMockFromSchema(schema.items)];

      case 'string':
        return schema.example || 'string';

      case 'number':
      case 'integer':
        return schema.example || 0;

      case 'boolean':
        return schema.example || false;

      default:
        return null;
    }
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(mockData);
  }
</script>

<div class="mock-data">
  <div class="sidebar">
    <h3>Endpoints</h3>
    <div class="endpoints-list">
      {#if spec === null}
        <div class="loading">Loading schemas...</div>
      {:else if spec}
        {#each Object.entries(spec.paths) as [path, methods]}
          <div class="endpoint">
            <div class="endpoint-path">{path}</div>
            <div class="methods">
              {#each Object.entries(methods) as [method, details]}
                <button
                  class="method-button"
                  class:selected={selectedPath === path && selectedMethod === method}
                  on:click={() => selectEndpoint(path, method)}
                  style="background: {getMethodColor(method)}">
                  {method.toUpperCase()}
                </button>
              {/each}
            </div>
          </div>
        {/each}
      {/if}
    </div>
  </div>

  <div class="main-content">
    {#if error}
      <div class="error">
        Error: {error}
      </div>
    {:else if selectedPath && selectedMethod && spec}
      <div class="mock-preview">
        <h2>
          <span class="method" style="background: {getMethodColor(selectedMethod)}">
            {selectedMethod.toUpperCase()}
          </span>
          {selectedPath}
        </h2>

        <div class="mock-actions">
          <button class="copy-btn" on:click={copyToClipboard}>
            Copy to Clipboard
          </button>
          <button class="refresh-btn" on:click={generateMockData}>
            Regenerate
          </button>
        </div>

        <pre class="mock-json">{mockData}</pre>
      </div>
    {:else}
      <div class="empty-state">
        Select an endpoint to generate mock data
      </div>
    {/if}
  </div>
</div>

<style>
  .mock-data {
    display: grid;
    grid-template-columns: 250px 1fr;
    gap: 1rem;
    height: 100%;
    overflow: hidden;
  }

  .sidebar {
    border-right: 1px solid #ddd;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .sidebar h3 {
    padding: 1rem;
    margin: 0;
    border-bottom: 1px solid #ddd;
  }

  .endpoints-list {
    overflow-y: auto;
    padding: 1rem;
  }

  .endpoint {
    margin-bottom: 1rem;
  }

  .endpoint-path {
    font-family: monospace;
    margin-bottom: 0.5rem;
    word-break: break-all;
  }

  .methods {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .method-button {
    border: none;
    padding: 0.25rem 0.5rem;
    border-radius: 3px;
    color: white;
    cursor: pointer;
    font-size: 0.8rem;
    opacity: 0.8;
  }

  .method-button:hover {
    opacity: 1;
  }

  .method-button.selected {
    opacity: 1;
    box-shadow: 0 0 0 2px white, 0 0 0 4px var(--method-color);
  }

  .main-content {
    overflow-y: auto;
    padding: 1rem;
  }

  .mock-preview h2 {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 2rem;
  }

  .method {
    padding: 0.25rem 0.5rem;
    border-radius: 3px;
    color: white;
    font-size: 0.9rem;
  }

  .mock-actions {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .copy-btn, .refresh-btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .copy-btn {
    background: #28a745;
    color: white;
  }

  .copy-btn:hover {
    background: #218838;
  }

  .refresh-btn {
    background: #6c757d;
    color: white;
  }

  .refresh-btn:hover {
    background: #5a6268;
  }

  .mock-json {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 4px;
    overflow: auto;
    font-family: monospace;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    font-style: italic;
  }

  .error {
    padding: 1rem;
    margin: 1rem;
    background: #f8d7da;
    border: 1px solid #f5c6cb;
    border-radius: 4px;
    color: #721c24;
  }

  .loading {
    padding: 1rem;
    color: #666;
    font-style: italic;
    text-align: center;
  }
</style>