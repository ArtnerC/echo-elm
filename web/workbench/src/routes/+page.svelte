<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Library, type ParityRun } from '$lib/api';

  let libraries: Library[] = [];
  let parityRuns: ParityRun[] = [];
  let loading = true;
  let error = '';

  onMount(async () => {
    try {
      const [libs, parity] = await Promise.all([
        api.libraries(),
        api.parityRuns()
      ]);
      libraries = libs.libraries;
      parityRuns = parity.runs;
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });
</script>

<div class="max-w-3xl mx-auto space-y-6">
  <div>
    <h1 class="text-xl font-bold text-brand-light mb-1">Library Browser</h1>
    <p class="text-slate-500 text-sm">CQL libraries in the workspace. Click any row to open in the translator.</p>
  </div>

  {#if loading}
    <div class="text-slate-500 text-sm">Loading…</div>
  {:else if error}
    <div class="card border-red-800">
      <p class="text-red-400 text-sm">Failed to load libraries: {error}</p>
    </div>
  {:else if libraries.length === 0}
    <div class="card">
      <p class="text-slate-500 text-sm">No CQL libraries found in the workspace.</p>
    </div>
  {:else}
    <div class="card p-0 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-surface-muted text-slate-400 text-xs">
          <tr>
            <th class="text-left px-4 py-2">Name @ Version</th>
            <th class="text-left px-4 py-2">Path</th>
            <th class="text-left px-4 py-2">Includes</th>
          </tr>
        </thead>
        <tbody>
          {#each libraries as lib, i}
            <tr
              class="border-t border-surface-border hover:bg-surface-muted cursor-pointer transition-colors"
              onclick={() => location.href = `/translate?path=${encodeURIComponent(lib.path)}`}
            >
              <td class="px-4 py-2 font-semibold text-brand-light">
                {lib.name}<span class="text-slate-500 font-normal">@{lib.version}</span>
              </td>
              <td class="px-4 py-2 text-slate-400">{lib.path}</td>
              <td class="px-4 py-2 text-slate-500">{lib.includes?.join(', ') ?? '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  {#if parityRuns.length > 0}
    <div>
      <h2 class="text-base font-semibold text-slate-300 mb-2">Parity Runs</h2>
      <div class="card p-0 overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-surface-muted text-slate-400 text-xs">
            <tr>
              <th class="text-left px-4 py-2">Run ID</th>
              <th class="text-left px-4 py-2">Created</th>
              <th class="text-left px-4 py-2">Fixtures</th>
              <th class="text-left px-4 py-2">Deltas</th>
            </tr>
          </thead>
          <tbody>
            {#each parityRuns as run}
              <tr
                class="border-t border-surface-border hover:bg-surface-muted cursor-pointer"
                onclick={() => location.href = `/parity/${run.id}`}
              >
                <td class="px-4 py-2 text-blue-400">{run.id}</td>
                <td class="px-4 py-2 text-slate-400">{new Date(run.createdAt).toLocaleString()}</td>
                <td class="px-4 py-2">{run.fixtureCount}</td>
                <td class="px-4 py-2 {run.deltaCount > 0 ? 'text-yellow-400' : 'text-green-400'}">{run.deltaCount}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>
