<script>
    import { onMount } from 'svelte';
    import { fetchResource, createResource, updateResource, deleteResource } from './api';

    export let path = '';
    export let schema = null;

    let resources = [];
    let loading = true;
    let error = null;
    let editingResource = null;
    let newResource = {};

    onMount(async () => {
        await loadResources();
    });

    async function loadResources() {
        try {
            loading = true;
            error = null;
            resources = await fetchResource(path);
        } catch (e) {
            error = e.message;
        } finally {
            loading = false;
        }
    }

    async function handleCreate() {
        try {
            error = null;
            await createResource(path, newResource);
            newResource = {};
            await loadResources();
        } catch (e) {
            error = e.message;
        }
    }

    async function handleUpdate(resource) {
        try {
            error = null;
            await updateResource(`${path}/${resource.id}`, editingResource);
            editingResource = null;
            await loadResources();
        } catch (e) {
            error = e.message;
        }
    }

    async function handleDelete(id) {
        try {
            error = null;
            await deleteResource(`${path}/${id}`);
            await loadResources();
        } catch (e) {
            error = e.message;
        }
    }

    function startEdit(resource) {
        editingResource = { ...resource };
    }

    function cancelEdit() {
        editingResource = null;
    }
</script>

<div class="resource-view">
    {#if loading}
        <div class="loading">Loading...</div>
    {:else if error}
        <div class="error">{error}</div>
    {:else}
        <div class="resource-list">
            <h3>Resources at {path}</h3>
            
            <!-- Create new resource -->
            <div class="create-form">
                <h4>Create New Resource</h4>
                {#if schema}
                    {#each Object.entries(schema.properties) as [key, prop]}
                        {#if key !== 'id'}
                            <div class="form-field">
                                <label for={`new-${key}`}>{key}</label>
                                <input 
                                    id={`new-${key}`}
                                    type="text"
                                    bind:value={newResource[key]}
                                    placeholder={prop.example || key}
                                />
                            </div>
                        {/if}
                    {/each}
                {/if}
                <button on:click={handleCreate}>Create</button>
            </div>

            <!-- Resource list -->
            {#each resources as resource (resource.id)}
                <div class="resource-item">
                    {#if editingResource && editingResource.id === resource.id}
                        <div class="edit-form">
                            {#each Object.entries(resource) as [key, value]}
                                {#if key !== 'id'}
                                    <div class="form-field">
                                        <label for={`edit-${key}-${resource.id}`}>{key}</label>
                                        <input 
                                            id={`edit-${key}-${resource.id}`}
                                            type="text"
                                            bind:value={editingResource[key]}
                                        />
                                    </div>
                                {/if}
                            {/each}
                            <div class="actions">
                                <button on:click={() => handleUpdate(resource)}>Save</button>
                                <button on:click={cancelEdit}>Cancel</button>
                            </div>
                        </div>
                    {:else}
                        <div class="resource-content">
                            {#each Object.entries(resource) as [key, value]}
                                <div class="field">
                                    <strong>{key}:</strong> {value}
                                </div>
                            {/each}
                            <div class="actions">
                                <button on:click={() => startEdit(resource)}>Edit</button>
                                <button on:click={() => handleDelete(resource.id)}>Delete</button>
                            </div>
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .resource-view {
        padding: 20px;
    }

    .loading, .error {
        padding: 20px;
        text-align: center;
    }

    .error {
        color: #dc3545;
        background: #f8d7da;
        border-radius: 4px;
    }

    .resource-list {
        max-width: 800px;
        margin: 0 auto;
    }

    .resource-item {
        background: white;
        border: 1px solid #dee2e6;
        border-radius: 4px;
        margin: 10px 0;
        padding: 15px;
    }

    .field {
        margin: 5px 0;
    }

    .actions {
        margin-top: 10px;
        display: flex;
        gap: 10px;
    }

    .form-field {
        margin: 10px 0;
    }

    .form-field label {
        display: block;
        margin-bottom: 5px;
        font-weight: bold;
    }

    .form-field input {
        width: 100%;
        padding: 8px;
        border: 1px solid #ced4da;
        border-radius: 4px;
    }

    button {
        padding: 8px 16px;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        background: #007bff;
        color: white;
    }

    button:hover {
        background: #0056b3;
    }

    .create-form {
        background: #f8f9fa;
        padding: 20px;
        border-radius: 4px;
        margin-bottom: 20px;
    }
</style> 