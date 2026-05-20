<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api, type ParityRun } from '$lib/api';
  import { marked } from 'marked';

  const id = $page.params.id;
  let run: ParityRun | null = null;
  let reportHtml = '';
  let loading = true;
  let error = '';
  let diffOpen: string | null = null;
  let diffContent = '';

  onMount(async () => {
    try {
      const [runData, md] = await Promise.all([
        api.parityRun(id),
        api.parityReportMd(id).catch(() => '')
      ]);
      run = runData.run;
      if (md) reportHtml = await marked(md);
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  });

  async function openDiff(fixture: string) {
    if (diffOpen === fixture) { diffOpen = null; return; }
    diffOpen = fixture;
    try {
      const res = await fetch(`/api/parity/runs/${encodeURIComponent(id)}/diffs/${encodeURIComponent(fixture)}`);
      diffContent = res.ok ? await res.text() : `Could not load diff: ${res.status}`;
    } catch (e) {
      diffContent = String(e);
    }
  }
</script>

<div class="max-w-3xl mx-auto space-y-4">
  <div class="flex items-center gap-3">
    <a href="/" class="text-slate-500 text-sm hover:text-slate-300">← Libraries</a>
    <h1 class="text-xl font-bold text-brand-light">Parity Run</h1>
    <code class="text-xs bg-surface-muted px-2 py-0.5 rounded text-slate-300">{id}</code>
  </div>

  {#if loading}
    <div class="text-slate-500 text-sm">Loading…</div>
  {:else if error}
    <div class="card border-red-800 text-red-400 text-sm">{error}</div>
  {:else if run}
    <div class="card flex gap-8 text-sm">
      <div><span class="text-slate-500">Created</span><br><span>{new Date(run.createdAt).toLocaleString()}</span></div>
      <div><span class="text-slate-500">Fixtures</span><br><span>{run.fixtureCount}</span></div>
      <div><span class="text-slate-500">Deltas</span><br>
        <span class="{run.deltaCount > 0 ? 'text-yellow-400' : 'text-green-400'}">{run.deltaCount}</span>
      </div>
    </div>

    {#if reportHtml}
      <div class="card prose prose-invert prose-sm max-w-none
        prose-headings:text-brand-light prose-a:text-blue-400
        prose-code:bg-surface-muted prose-code:px-1 prose-code:rounded">
        {@html reportHtml}
      </div>
    {:else}
      <div class="card text-slate-500 text-sm">No report.md found for this run.</div>
    {/if}

    {#if run.deltaCount > 0}
      <div>
        <h2 class="text-sm font-semibold text-slate-300 mb-2">Diffs</h2>
        <p class="text-slate-500 text-xs mb-2">Click a fixture name to expand its diff.</p>
        <!-- Fixture diff list — populated when parity runs exist -->
        <div class="card text-slate-500 text-sm">
          {#if diffOpen}
            <pre class="text-xs overflow-auto max-h-80 text-slate-300 whitespace-pre">{diffContent}</pre>
          {:else}
            <span>Select a fixture above to view its diff.</span>
          {/if}
        </div>
      </div>
    {/if}
  {/if}
</div>
