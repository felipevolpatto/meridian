<script>
  import { onMount } from 'svelte';
  import { getOpenAPISpec, getServerStatus, setHistoryCallback } from './lib/api';
  import StateInspector from './lib/StateInspector.svelte';
  import APITester from './lib/APITester.svelte';
  import RequestHistory from './lib/RequestHistory.svelte';
  import APIDocs from './lib/APIDocs.svelte';
  import RequestBuilder from './lib/RequestBuilder.svelte';
  import MockData from './lib/MockData.svelte';

  let spec = null;
  let status = null;
  let error = null;
  let activeTab = 'state';
  let requestHistoryComponent;

  onMount(async () => {
    try {
      [spec, status] = await Promise.all([
        getOpenAPISpec(),
        getServerStatus()
      ]);

      // Set up request history tracking
      setHistoryCallback((request, response) => {
        if (requestHistoryComponent) {
          requestHistoryComponent.addHistoryEntry(request, response);
        }
      });
    } catch (err) {
      error = err.message;
    }
  });
</script>

<main>
  <header>
    <h1>Meridian Mock Server</h1>
    {#if status}
      <div class="status">
        <span class="status-indicator" class:online={status.status === 'online'}>
          {status.status}
        </span>
        {#if status.version}
          <span class="version">v{status.version}</span>
        {/if}
      </div>
    {/if}
  </header>

  {#if error}
    <div class="error">
      Error: {error}
    </div>
  {/if}

  <nav>
    <ul>
      <li>
        <button
          class:active={activeTab === 'state'}
          on:click={() => activeTab = 'state'}>
          State Inspector
        </button>
      </li>
      <li>
        <button
          class:active={activeTab === 'api'}
          on:click={() => activeTab = 'api'}>
          API Testing
        </button>
      </li>
      <li>
        <button
          class:active={activeTab === 'history'}
          on:click={() => activeTab = 'history'}>
          Request History
        </button>
      </li>
      <li>
        <button
          class:active={activeTab === 'docs'}
          on:click={() => activeTab = 'docs'}>
          API Documentation
        </button>
      </li>
      <li>
        <button
          class:active={activeTab === 'builder'}
          on:click={() => activeTab = 'builder'}>
          Request Builder
        </button>
      </li>
      <li>
        <button
          class:active={activeTab === 'mock'}
          on:click={() => activeTab = 'mock'}>
          Mock Data
        </button>
      </li>
    </ul>
  </nav>

  <div class="content">
    {#if activeTab === 'state'}
      <StateInspector />
    {:else if activeTab === 'api'}
      <APITester />
    {:else if activeTab === 'history'}
      <RequestHistory bind:this={requestHistoryComponent} />
    {:else if activeTab === 'docs'}
      <APIDocs />
    {:else if activeTab === 'builder'}
      <RequestBuilder />
    {:else if activeTab === 'mock'}
      <MockData />
    {/if}
  </div>
</main>

<style>
  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 1rem;
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .status-indicator {
    display: inline-block;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    background: #dc3545;
    color: white;
  }

  .status-indicator.online {
    background: #28a745;
  }

  .version {
    color: #666;
  }

  nav {
    margin-bottom: 1rem;
  }

  nav ul {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    gap: 1rem;
    border-bottom: 1px solid #ddd;
  }

  nav button {
    padding: 0.5rem 1rem;
    border: none;
    background: none;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }

  nav button:hover {
    color: #007bff;
  }

  nav button.active {
    border-bottom-color: #007bff;
    color: #007bff;
  }

  .content {
    flex: 1;
    overflow: hidden;
    background: white;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  .error {
    padding: 1rem;
    margin-bottom: 1rem;
    background: #f8d7da;
    border: 1px solid #f5c6cb;
    border-radius: 4px;
    color: #721c24;
  }
</style>
