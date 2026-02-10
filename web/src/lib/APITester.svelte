<!-- APITester.svelte -->
<script>
  import { onMount } from 'svelte';
  import { getOpenAPISpec, generateExample } from './api';

  let spec = null;
  let error = null;
  let loading = true;
  let selectedPath = null;
  let selectedMethod = null;
  let requestBody = '';
  let response = null;
  let responseStatus = null;
  let responseTime = null;

  onMount(async () => {
    try {
      spec = await getOpenAPISpec();
      loading = false;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  });

  function getMethodColor(method) {
    switch (method.toLowerCase()) {
      case 'get': return '#e3f2fd';
      case 'post': return '#e8f5e9';
      case 'put': return '#fff3e0';
      case 'delete': return '#ffebee';
      default: return '#f5f5f5';
    }
  }

  function getMethodTextColor(method) {
    switch (method.toLowerCase()) {
      case 'get': return '#1565c0';
      case 'post': return '#2e7d32';
      case 'put': return '#f57c00';
      case 'delete': return '#c62828';
      default: return '#333333';
    }
  }

  async function generateExampleBody() {
    if (!selectedPath || !selectedMethod) return;

    const operation = spec.paths[selectedPath][selectedMethod.toLowerCase()];
    if (!operation?.requestBody?.content?.['application/json']?.schema) return;

    try {
      const example = await generateExample(selectedPath.split('/')[1]);
      requestBody = JSON.stringify(example, null, 2);
    } catch (err) {
      error = err.message;
    }
  }

  async function sendRequest() {
    if (!selectedPath || !selectedMethod) return;

    error = null;
    response = null;
    responseStatus = null;
    responseTime = null;

    const startTime = performance.now();

    try {
      const url = `/api${selectedPath}`;
      const options = {
        method: selectedMethod,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
      };

      if (requestBody && ['POST', 'PUT', 'PATCH'].includes(selectedMethod)) {
        options.body = requestBody;
      }

      const res = await fetch(url, options);
      responseStatus = res.status;
      responseTime = Math.round(performance.now() - startTime);

      if (res.headers.get('Content-Type')?.includes('application/json')) {
        response = await res.json();
      } else {
        response = await res.text();
      }
    } catch (err) {
      error = err.message;
    }
  }
</script>

<div class="api-tester">
  <div class="header">
    <h2>API Testing</h2>
  </div>

  {#if error}
    <div class="error">
      Error: {error}
    </div>
  {/if}

  {#if loading}
    <div class="loading">Loading OpenAPI specification...</div>
  {:else if spec}
    <div class="endpoints">
      <div class="endpoint-selector">
        <h3>Endpoints</h3>
        <select bind:value={selectedPath}>
          <option value={null}>Select an endpoint...</option>
          {#each Object.entries(spec.paths) as [path, methods]}
            <option value={path}>{path}</option>
          {/each}
        </select>
      </div>

      {#if selectedPath}
        <div class="method-selector">
          <h3>Method</h3>
          <div class="methods">
            {#each Object.keys(spec.paths[selectedPath]) as method}
              {#if !method.startsWith('x-')}
                <button
                  class="method"
                  class:selected={selectedMethod === method.toUpperCase()}
                  style="background: {getMethodColor(method)}; color: {getMethodTextColor(method)}"
                  on:click={() => selectedMethod = method.toUpperCase()}>
                  {method.toUpperCase()}
                </button>
              {/if}
            {/each}
          </div>
        </div>

        {#if selectedMethod}
          <div class="request-body">
            <div class="request-header">
              <h3>Request Body</h3>
              {#if ['POST', 'PUT', 'PATCH'].includes(selectedMethod)}
                <button class="generate-btn" on:click={generateExampleBody}>
                  Generate Example
                </button>
              {/if}
            </div>
            <textarea
              bind:value={requestBody}
              placeholder="Enter request body (JSON)"
              rows="10"
              disabled={!['POST', 'PUT', 'PATCH'].includes(selectedMethod)} />
          </div>

          <div class="actions">
            <button class="send-btn" on:click={sendRequest}>
              Send Request
            </button>
          </div>

          {#if response !== null}
            <div class="response">
              <div class="response-header">
                <h3>Response</h3>
                <div class="response-meta">
                  {#if responseStatus}
                    <span class="status" class:success={responseStatus < 400}>
                      {responseStatus}
                    </span>
                  {/if}
                  {#if responseTime}
                    <span class="time">{responseTime}ms</span>
                  {/if}
                </div>
              </div>
              <pre>{typeof response === 'string' ? response : JSON.stringify(response, null, 2)}</pre>
            </div>
          {/if}
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .api-tester {
    padding: 1rem;
  }

  .header {
    margin-bottom: 1rem;
  }

  .error {
    color: red;
    padding: 1rem;
    margin-bottom: 1rem;
    background-color: #ffebee;
    border-radius: 4px;
  }

  .loading {
    padding: 2rem;
    text-align: center;
    color: #666;
  }

  .endpoints {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .endpoint-selector select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
  }

  .methods {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .method {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    font-weight: bold;
    cursor: pointer;
    transition: opacity 0.2s;
  }

  .method:hover {
    opacity: 0.8;
  }

  .method.selected {
    outline: 2px solid currentColor;
  }

  .request-body {
    background-color: #f5f5f5;
    padding: 1rem;
    border-radius: 4px;
  }

  .request-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .generate-btn {
    padding: 0.25rem 0.5rem;
    background-color: #2196f3;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .generate-btn:hover {
    background-color: #1976d2;
  }

  textarea {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-family: monospace;
    resize: vertical;
  }

  textarea:disabled {
    background-color: #eee;
    cursor: not-allowed;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: 1rem;
  }

  .send-btn {
    padding: 0.5rem 1.5rem;
    background-color: #4caf50;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
  }

  .send-btn:hover {
    background-color: #45a049;
  }

  .response {
    background-color: #f5f5f5;
    padding: 1rem;
    border-radius: 4px;
    margin-top: 1rem;
  }

  .response-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }

  .response-meta {
    display: flex;
    gap: 1rem;
    align-items: center;
  }

  .status {
    padding: 0.25rem 0.5rem;
    background-color: #f44336;
    color: white;
    border-radius: 4px;
    font-weight: bold;
  }

  .status.success {
    background-color: #4caf50;
  }

  .time {
    color: #666;
    font-size: 0.9rem;
  }

  pre {
    margin: 0;
    padding: 1rem;
    background-color: white;
    border-radius: 4px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
</style> 