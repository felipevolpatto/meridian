<!-- StateInspector.svelte -->
<script>
  import { onMount, onDestroy } from 'svelte';
  import { fetchState, deleteResource, updateResource } from './api';
  import { subscribe, getConnectionStatus } from './websocket';

  let resources = {};
  let selectedType = '';
  let selectedResource = null;
  let editMode = false;
  let editedData = '';
  let error = '';
  let searchTerm = '';
  let filterType = 'all';
  let sortOrder = 'asc';
  let expandedNodes = new Set();
  let connectionStatus = getConnectionStatus();
  let unsubscribe;

  $: filteredResources = filterAndSearchResources(resources, searchTerm, filterType);
  $: sortedResources = sortResources(filteredResources, sortOrder);

  onMount(async () => {
    try {
      await refreshState();
      
      // Subscribe to WebSocket updates
      unsubscribe = subscribe(handleWebSocketMessage);
    } catch (err) {
      error = err.message;
    }
  });

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe();
    }
  });

  function handleWebSocketMessage(message) {
    switch (message.type) {
      case 'state_update':
        refreshState();
        break;
      case 'connection':
        connectionStatus = message.status;
        if (message.status === 'connected') {
          refreshState();
        }
        break;
      case 'error':
        error = 'WebSocket error: Connection interrupted';
        break;
    }
  }

  async function refreshState() {
    try {
      resources = await fetchState();
      error = '';
    } catch (err) {
      error = err.message;
    }
  }

  function filterAndSearchResources(resources, search, type) {
    let filtered = { ...resources };
    
    if (type !== 'all') {
      filtered = {
        [type]: resources[type]
      };
    }

    if (search.trim()) {
      const searchLower = search.toLowerCase();
      Object.keys(filtered).forEach(key => {
        filtered[key] = filtered[key].filter(resource => 
          JSON.stringify(resource).toLowerCase().includes(searchLower)
        );
      });
    }

    return filtered;
  }

  function sortResources(resources, order) {
    const sorted = { ...resources };
    Object.keys(sorted).forEach(key => {
      sorted[key] = [...sorted[key]].sort((a, b) => {
        const aStr = JSON.stringify(a);
        const bStr = JSON.stringify(b);
        return order === 'asc' ? 
          aStr.localeCompare(bStr) : 
          bStr.localeCompare(aStr);
      });
    });
    return sorted;
  }

  function toggleNode(id) {
    if (expandedNodes.has(id)) {
      expandedNodes.delete(id);
    } else {
      expandedNodes.add(id);
    }
    expandedNodes = expandedNodes;
  }

  function formatValue(value) {
    if (typeof value === 'object' && value !== null) {
      return JSON.stringify(value, null, 2);
    }
    return String(value);
  }

  function selectResource(resource) {
    selectedResource = resource;
    selectedType = Object.entries(resources).find(([type, items]) => 
      items.some(item => item.id === resource.id)
    )?.[0];
    editMode = false;
  }

  function toggleEditMode() {
    if (!editMode) {
      editedData = JSON.stringify(selectedResource, null, 2);
    }
    editMode = !editMode;
  }

  async function saveChanges() {
    try {
      let parsedData = JSON.parse(editedData);
      await updateResource(selectedType, selectedResource.id, parsedData);
      editMode = false;
      error = '';
    } catch (err) {
      error = `Failed to save changes: ${err.message}`;
    }
  }

  async function deleteSelected() {
    if (!selectedResource || !selectedType) return;
    
    try {
      await deleteResource(selectedType, selectedResource.id);
      selectedResource = null;
      error = '';
    } catch (err) {
      error = `Failed to delete resource: ${err.message}`;
    }
  }
</script>

<div class="state-inspector">
  <div class="toolbar">
    <div class="connection-status" class:connected={connectionStatus === 'connected'}>
      <span class="status-indicator"></span>
      {connectionStatus}
    </div>

    <input
      type="text"
      bind:value={searchTerm}
      placeholder="Search resources..."
      class="search-input"
    />
    
    <select bind:value={filterType} class="filter-select">
      <option value="all">All Types</option>
      {#each Object.keys(resources) as type}
        <option value={type}>{type}</option>
      {/each}
    </select>

    <select bind:value={sortOrder} class="sort-select">
      <option value="asc">Sort A-Z</option>
      <option value="desc">Sort Z-A</option>
    </select>
  </div>

  {#if error}
    <div class="error">
      {error}
    </div>
  {/if}

  <div class="resources-container">
    {#each Object.entries(sortedResources) as [type, typeResources]}
      <div class="resource-type">
        <h3>{type} ({typeResources.length})</h3>
        <div class="resource-list">
          {#each typeResources as resource}
            <div 
              class="resource-item"
              class:selected={selectedResource?.id === resource.id}
              on:click={() => selectResource(resource)}
            >
              <div class="resource-header">
                <span class="resource-id">ID: {resource.id}</span>
                <button 
                  class="toggle-btn"
                  on:click|stopPropagation={() => toggleNode(resource.id)}
                >
                  {expandedNodes.has(resource.id) ? '▼' : '▶'}
                </button>
              </div>

              {#if expandedNodes.has(resource.id)}
                <div class="resource-preview">
                  {#each Object.entries(resource) as [key, value]}
                    <div class="resource-field">
                      <span class="field-name">{key}:</span>
                      <span class="field-value">{formatValue(value)}</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/each}
  </div>

  {#if selectedResource}
    <div class="resource-detail">
      <div class="detail-header">
        <h3>Selected Resource</h3>
        <div class="detail-actions">
          <button on:click={toggleEditMode}>
            {editMode ? 'Cancel' : 'Edit'}
          </button>
          {#if editMode}
            <button on:click={saveChanges} class="save-btn">Save</button>
          {/if}
          <button on:click={deleteSelected} class="delete-btn">Delete</button>
        </div>
      </div>

      {#if editMode}
        <textarea
          bind:value={editedData}
          class="edit-textarea"
          rows="10"
        />
      {:else}
        <pre class="json-viewer">{JSON.stringify(selectedResource, null, 2)}</pre>
      {/if}
    </div>
  {/if}
</div>

<style>
  .state-inspector {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: 100%;
    padding: 1rem;
  }

  .toolbar {
    display: flex;
    gap: 1rem;
    padding: 0.5rem;
    background: #f5f5f5;
    border-radius: 4px;
  }

  .connection-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    background: #f8f9fa;
    border-radius: 4px;
    font-size: 0.9em;
    color: #dc3545;
  }

  .connection-status.connected {
    color: #28a745;
  }

  .status-indicator {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #dc3545;
  }

  .connection-status.connected .status-indicator {
    background: #28a745;
  }

  .search-input,
  .filter-select,
  .sort-select {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  .search-input {
    flex: 1;
  }

  .resources-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    overflow-y: auto;
  }

  .resource-type {
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 1rem;
  }

  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .resource-item {
    padding: 0.5rem;
    border: 1px solid #eee;
    border-radius: 4px;
    cursor: pointer;
  }

  .resource-item:hover {
    background: #f5f5f5;
  }

  .resource-item.selected {
    border-color: #007bff;
    background: #e7f1ff;
  }

  .resource-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .toggle-btn {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0 0.5rem;
  }

  .resource-preview {
    margin-top: 0.5rem;
    padding: 0.5rem;
    background: #f8f9fa;
    border-radius: 4px;
  }

  .resource-field {
    display: flex;
    gap: 0.5rem;
    font-size: 0.9em;
  }

  .field-name {
    font-weight: bold;
    color: #666;
  }

  .resource-detail {
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 1rem;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .detail-actions {
    display: flex;
    gap: 0.5rem;
  }

  .edit-textarea {
    width: 100%;
    font-family: monospace;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  .json-viewer {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 4px;
    overflow: auto;
    font-family: monospace;
  }

  .error {
    color: #dc3545;
    padding: 0.5rem;
    background: #f8d7da;
    border: 1px solid #f5c6cb;
    border-radius: 4px;
  }

  button {
    padding: 0.5rem 1rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    cursor: pointer;
  }

  button:hover {
    background: #f5f5f5;
  }

  .save-btn {
    background: #28a745;
    color: white;
    border-color: #28a745;
  }

  .save-btn:hover {
    background: #218838;
  }

  .delete-btn {
    background: #dc3545;
    color: white;
    border-color: #dc3545;
  }

  .delete-btn:hover {
    background: #c82333;
  }
</style> 