<!-- RequestBuilder.svelte -->
<script>
  import { onMount } from 'svelte';
  import { getOpenAPISpec } from './api';

  let spec = null;
  let error = null;
  let selectedPath = null;
  let selectedMethod = null;
  let formData = {};
  let queryParams = {};
  let pathParams = {};
  let headers = {};
  let requestBody = '';
  let response = null;
  let isLoading = false;

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

  function resetForm() {
    formData = {};
    queryParams = {};
    pathParams = {};
    headers = {};
    requestBody = '';
    response = null;
  }

  function selectEndpoint(path, method) {
    selectedPath = path;
    selectedMethod = method;
    resetForm();
  }

  function getDefaultValue(schema) {
    switch (schema.type) {
      case 'string':
        return schema.example || '';
      case 'number':
      case 'integer':
        return schema.example || 0;
      case 'boolean':
        return schema.example || false;
      case 'array':
        return schema.example || [];
      case 'object':
        return schema.example || {};
      default:
        return '';
    }
  }

  function buildUrl() {
    let url = selectedPath;
    
    // Replace path parameters
    Object.entries(pathParams).forEach(([key, value]) => {
      url = url.replace(`{${key}}`, encodeURIComponent(value));
    });

    // Add query parameters
    const query = Object.entries(queryParams)
      .filter(([_, value]) => value !== '')
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
      .join('&');

    if (query) {
      url = `${url}?${query}`;
    }

    return url;
  }

  async function sendRequest() {
    if (!selectedPath || !selectedMethod) return;

    const endpoint = spec.paths[selectedPath][selectedMethod];
    const url = buildUrl();
    
    const options = {
      method: selectedMethod.toUpperCase(),
      headers: {
        ...headers,
        'Content-Type': 'application/json'
      }
    };

    if (requestBody) {
      options.body = requestBody;
    }

    isLoading = true;
    response = null;

    try {
      const res = await fetch(url, options);
      const contentType = res.headers.get('content-type');
      
      response = {
        status: res.status,
        statusText: res.statusText,
        headers: Object.fromEntries(res.headers.entries()),
        body: contentType?.includes('application/json') 
          ? await res.json()
          : await res.text()
      };
    } catch (err) {
      response = {
        error: err.message
      };
    } finally {
      isLoading = false;
    }
  }

  function formatResponse(response) {
    if (!response) return '';
    
    if (response.error) {
      return `Error: ${response.error}`;
    }

    let formatted = `Status: ${response.status} ${response.statusText}\n\n`;
    formatted += 'Headers:\n';
    Object.entries(response.headers).forEach(([key, value]) => {
      formatted += `${key}: ${value}\n`;
    });
    formatted += '\nBody:\n';
    
    if (typeof response.body === 'object') {
      formatted += JSON.stringify(response.body, null, 2);
    } else {
      formatted += response.body;
    }

    return formatted;
  }
</script>

<div class="request-builder">
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
      {@const endpoint = spec.paths[selectedPath][selectedMethod]}
      <div class="request-form">
        <h2>
          <span class="method" style="background: {getMethodColor(selectedMethod)}">
            {selectedMethod.toUpperCase()}
          </span>
          {selectedPath}
        </h2>

        {#if endpoint.parameters}
          <div class="parameters">
            {#each endpoint.parameters.filter(p => p.in === 'path') as param}
              <div class="form-group">
                <label for={param.name}>
                  {param.name}
                  {#if param.required}*{/if}
                  <span class="param-type">path parameter</span>
                </label>
                <input
                  type="text"
                  id={param.name}
                  bind:value={pathParams[param.name]}
                  placeholder={param.description}
                  required={param.required}
                />
              </div>
            {/each}

            {#each endpoint.parameters.filter(p => p.in === 'query') as param}
              <div class="form-group">
                <label for={param.name}>
                  {param.name}
                  {#if param.required}*{/if}
                  <span class="param-type">query parameter</span>
                </label>
                <input
                  type="text"
                  id={param.name}
                  bind:value={queryParams[param.name]}
                  placeholder={param.description}
                  required={param.required}
                />
              </div>
            {/each}

            {#each endpoint.parameters.filter(p => p.in === 'header') as param}
              <div class="form-group">
                <label for={param.name}>
                  {param.name}
                  {#if param.required}*{/if}
                  <span class="param-type">header</span>
                </label>
                <input
                  type="text"
                  id={param.name}
                  bind:value={headers[param.name]}
                  placeholder={param.description}
                  required={param.required}
                />
              </div>
            {/each}
          </div>
        {/if}

        {#if endpoint.requestBody}
          <div class="request-body">
            <h3>Request Body</h3>
            {#each Object.entries(endpoint.requestBody.content) as [contentType, content]}
              <div class="content-type">
                <h4>{contentType}</h4>
                {#if contentType.includes('json')}
                  <textarea
                    bind:value={requestBody}
                    rows="10"
                    placeholder="Enter JSON request body"
                  />
                  {#if content.schema.example}
                    <button
                      class="example-btn"
                      on:click={() => requestBody = JSON.stringify(content.schema.example, null, 2)}>
                      Load Example
                    </button>
                  {/if}
                {:else}
                  <textarea
                    bind:value={requestBody}
                    rows="10"
                    placeholder="Enter request body"
                  />
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        <div class="actions">
          <button class="send-btn" on:click={sendRequest} disabled={isLoading}>
            {isLoading ? 'Sending...' : 'Send Request'}
          </button>
          <button class="reset-btn" on:click={resetForm}>
            Reset
          </button>
        </div>

        {#if response}
          <div class="response">
            <h3>Response</h3>
            <pre>{formatResponse(response)}</pre>
          </div>
        {/if}
      </div>
    {:else}
      <div class="empty-state">
        Select an endpoint to start building a request
      </div>
    {/if}
  </div>
</div>

<style>
  .request-builder {
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

  .request-form h2 {
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

  .parameters {
    margin-bottom: 2rem;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: bold;
  }

  .param-type {
    font-weight: normal;
    color: #666;
    font-size: 0.9em;
    margin-left: 0.5rem;
  }

  input, textarea {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-family: monospace;
  }

  textarea {
    resize: vertical;
  }

  .content-type {
    margin-bottom: 1rem;
  }

  .content-type h4 {
    color: #666;
    margin-bottom: 0.5rem;
  }

  .actions {
    display: flex;
    gap: 1rem;
    margin: 2rem 0;
  }

  .send-btn, .reset-btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .send-btn {
    background: #28a745;
    color: white;
  }

  .send-btn:hover {
    background: #218838;
  }

  .send-btn:disabled {
    background: #6c757d;
    cursor: not-allowed;
  }

  .reset-btn {
    background: #6c757d;
    color: white;
  }

  .reset-btn:hover {
    background: #5a6268;
  }

  .example-btn {
    margin-top: 0.5rem;
    padding: 0.25rem 0.5rem;
    background: #f8f9fa;
    border: 1px solid #ddd;
    border-radius: 4px;
    cursor: pointer;
  }

  .example-btn:hover {
    background: #e9ecef;
  }

  .response {
    margin-top: 2rem;
  }

  .response pre {
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