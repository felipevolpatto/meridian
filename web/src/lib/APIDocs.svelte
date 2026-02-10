<!-- APIDocs.svelte -->
<script>
  import { onMount } from 'svelte';
  import { getOpenAPISpec } from './api';

  let spec = null;
  let error = null;
  let selectedPath = null;
  let selectedMethod = null;
  let searchTerm = '';
  let expandedSchemas = new Set();

  onMount(async () => {
    try {
      spec = await getOpenAPISpec();
    } catch (err) {
      error = err.message;
    }
  });

  $: filteredPaths = spec && filterPaths(spec.paths, searchTerm);

  function filterPaths(paths, search) {
    if (!search.trim()) return paths;

    const searchLower = search.toLowerCase();
    const filtered = {};

    Object.entries(paths).forEach(([path, methods]) => {
      if (path.toLowerCase().includes(searchLower)) {
        filtered[path] = methods;
        return;
      }

      const matchingMethods = {};
      Object.entries(methods).forEach(([method, details]) => {
        if (
          method.toLowerCase().includes(searchLower) ||
          details.summary?.toLowerCase().includes(searchLower) ||
          details.description?.toLowerCase().includes(searchLower)
        ) {
          matchingMethods[method] = details;
        }
      });

      if (Object.keys(matchingMethods).length > 0) {
        filtered[path] = matchingMethods;
      }
    });

    return filtered;
  }

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

  function formatRef(ref) {
    if (!ref) return '';
    return ref.replace('#/components/schemas/', '');
  }

  function toggleSchema(name) {
    if (expandedSchemas.has(name)) {
      expandedSchemas.delete(name);
    } else {
      expandedSchemas.add(name);
    }
    expandedSchemas = expandedSchemas;
  }

  function renderSchema(schema, level = 0) {
    if (!schema) return '';

    if (schema.$ref) {
      const refName = formatRef(schema.$ref);
      const refSchema = spec.components?.schemas?.[refName];
      return renderSchema(refSchema, level);
    }

    if (schema.type === 'object') {
      return renderObjectSchema(schema, level);
    }

    if (schema.type === 'array') {
      return `Array of ${renderSchema(schema.items, level)}`;
    }

    return schema.type;
  }

  function renderObjectSchema(schema, level = 0) {
    if (!schema.properties) return 'object';

    const indent = '  '.repeat(level);
    let result = 'object {\n';

    Object.entries(schema.properties).forEach(([prop, propSchema]) => {
      const required = schema.required?.includes(prop) ? '*' : '';
      result += `${indent}  ${prop}${required}: ${renderSchema(propSchema, level + 1)}\n`;
    });

    result += `${indent}}`;
    return result;
  }
</script>

<div class="api-docs">
  {#if error}
    <div class="error">
      Error loading API documentation: {error}
    </div>
  {:else if spec}
    <div class="sidebar">
      <div class="search-box">
        <input
          type="text"
          bind:value={searchTerm}
          placeholder="Search endpoints..."
          class="search-input"
        />
      </div>

      <div class="endpoints-list">
        {#each Object.entries(filteredPaths) as [path, methods]}
          <div class="endpoint">
            <div class="endpoint-path">{path}</div>
            <div class="methods">
              {#each Object.entries(methods) as [method, details]}
                <button
                  class="method-button"
                  class:selected={selectedPath === path && selectedMethod === method}
                  on:click={() => {
                    selectedPath = path;
                    selectedMethod = method;
                  }}
                  style="background: {getMethodColor(method)}">
                  {method.toUpperCase()}
                </button>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    </div>

    <div class="main-content">
      {#if selectedPath && selectedMethod}
        {@const endpoint = spec.paths[selectedPath][selectedMethod]}
        <div class="endpoint-details">
          <h2>
            <span class="method" style="background: {getMethodColor(selectedMethod)}">
              {selectedMethod.toUpperCase()}
            </span>
            {selectedPath}
          </h2>

          {#if endpoint.summary}
            <div class="summary">{endpoint.summary}</div>
          {/if}

          {#if endpoint.description}
            <div class="description">{endpoint.description}</div>
          {/if}

          {#if endpoint.parameters?.length}
            <section>
              <h3>Parameters</h3>
              <table>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Location</th>
                    <th>Type</th>
                    <th>Required</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  {#each endpoint.parameters as param}
                    <tr>
                      <td>{param.name}</td>
                      <td>{param.in}</td>
                      <td>{renderSchema(param.schema)}</td>
                      <td>{param.required ? 'Yes' : 'No'}</td>
                      <td>{param.description || ''}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </section>
          {/if}

          {#if endpoint.requestBody}
            <section>
              <h3>Request Body</h3>
              {#each Object.entries(endpoint.requestBody.content) as [contentType, content]}
                <div class="content-type">
                  <h4>{contentType}</h4>
                  <pre class="schema">{renderSchema(content.schema)}</pre>
                </div>
              {/each}
            </section>
          {/if}

          {#if endpoint.responses}
            <section>
              <h3>Responses</h3>
              {#each Object.entries(endpoint.responses) as [status, response]}
                <div class="response">
                  <h4>
                    <span class="status-code">{status}</span>
                    {response.description}
                  </h4>
                  {#if response.content}
                    {#each Object.entries(response.content) as [contentType, content]}
                      <div class="content-type">
                        <h5>{contentType}</h5>
                        <pre class="schema">{renderSchema(content.schema)}</pre>
                      </div>
                    {/each}
                  {/if}
                </div>
              {/each}
            </section>
          {/if}
        </div>
      {:else}
        <div class="empty-state">
          Select an endpoint to view its documentation
        </div>
      {/if}
    </div>

    {#if spec.components?.schemas}
      <div class="schemas-sidebar">
        <h3>Schemas</h3>
        {#each Object.entries(spec.components.schemas) as [name, schema]}
          <div class="schema-item">
            <button
              class="schema-toggle"
              on:click={() => toggleSchema(name)}
            >
              {expandedSchemas.has(name) ? '▼' : '▶'} {name}
            </button>
            {#if expandedSchemas.has(name)}
              <pre class="schema">{renderSchema(schema)}</pre>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {:else}
    <div class="loading">
      Loading API documentation...
    </div>
  {/if}
</div>

<style>
  .api-docs {
    display: grid;
    grid-template-columns: 250px 1fr 250px;
    gap: 1rem;
    height: 100%;
    overflow: hidden;
  }

  .sidebar {
    border-right: 1px solid #ddd;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .search-box {
    padding: 1rem;
    border-bottom: 1px solid #ddd;
  }

  .search-input {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
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

  .endpoint-details h2 {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .method {
    padding: 0.25rem 0.5rem;
    border-radius: 3px;
    color: white;
    font-size: 0.9rem;
  }

  .summary {
    font-size: 1.2rem;
    margin-bottom: 1rem;
  }

  .description {
    color: #666;
    margin-bottom: 2rem;
  }

  section {
    margin-bottom: 2rem;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 1rem;
  }

  th, td {
    padding: 0.5rem;
    text-align: left;
    border: 1px solid #ddd;
  }

  th {
    background: #f5f5f5;
  }

  .content-type {
    margin-bottom: 1rem;
  }

  .content-type h4, .content-type h5 {
    color: #666;
    margin-bottom: 0.5rem;
  }

  .schema {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 4px;
    overflow: auto;
    font-family: monospace;
    margin: 0;
  }

  .response {
    margin-bottom: 1rem;
  }

  .status-code {
    padding: 0.25rem 0.5rem;
    border-radius: 3px;
    background: #f5f5f5;
    margin-right: 0.5rem;
  }

  .schemas-sidebar {
    border-left: 1px solid #ddd;
    padding: 1rem;
    overflow-y: auto;
  }

  .schema-item {
    margin-bottom: 1rem;
  }

  .schema-toggle {
    width: 100%;
    text-align: left;
    padding: 0.5rem;
    background: none;
    border: 1px solid #ddd;
    border-radius: 4px;
    cursor: pointer;
  }

  .schema-toggle:hover {
    background: #f5f5f5;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    font-style: italic;
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
  }

  .error {
    padding: 1rem;
    margin: 1rem;
    background: #f8d7da;
    border: 1px solid #f5c6cb;
    border-radius: 4px;
    color: #721c24;
  }
</style> 