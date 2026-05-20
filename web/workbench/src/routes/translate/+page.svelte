<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { api, type Diagnostic, type TranslateResult } from '$lib/api';
  import { locatorHighlightField, jumpToLocator } from '$lib/cql-locator';
  import type { EditorView } from '@codemirror/view';

  // ── Editor state ────────────────────────────────────────────────────────────
  let editorEl: HTMLDivElement;
  let view: EditorView | null = null;
  let editorContent = `library Demo version '1.0.0'\n\nusing FHIR version '4.0.1'\n\ncontext Patient\n\ndefine "InitialPopulation": true\n`;

  // ── Options ─────────────────────────────────────────────────────────────────
  let format: 'both' | 'xml' | 'json' = 'both';
  let annotations = true;
  let locators = true;
  let signatureLevel: 'None' | 'Differing' | 'Overloads' | 'All' = 'Overloads';

  // ── Results ─────────────────────────────────────────────────────────────────
  let result: TranslateResult | null = null;
  let running = false;
  let runError = '';
  let activeTab: 'xml' | 'json' | 'diag' = 'xml';

  // ── Prefill from ?path= ──────────────────────────────────────────────────────
  onMount(async () => {
    const { EditorView, keymap, lineNumbers, highlightActiveLine } = await import('@codemirror/view');
    const { EditorState } = await import('@codemirror/state');
    const { defaultKeymap, historyKeymap } = await import('@codemirror/commands');
    const { history } = await import('@codemirror/commands');
    const { oneDark } = await import('@codemirror/theme-one-dark');
    const { syntaxHighlighting, defaultHighlightStyle, indentUnit } = await import('@codemirror/language');

    const pathParam = $page.url.searchParams.get('path');
    if (pathParam) {
      try {
        const lib = await api.library(pathParam);
        editorContent = lib.content;
      } catch {
        // leave default
      }
    }

    view = new EditorView({
      doc: editorContent,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        oneDark,
        syntaxHighlighting(defaultHighlightStyle),
        indentUnit.of('  '),
        locatorHighlightField,
        EditorView.theme({
          '&': { height: '320px', fontSize: '13px' },
          '.cm-scroller': { overflow: 'auto' },
          '.cm-locator-highlight': { backgroundColor: 'rgba(124,58,237,0.35)', borderRadius: '2px' }
        }),
        EditorView.updateListener.of(upd => {
          if (upd.docChanged) editorContent = upd.state.doc.toString();
        })
      ],
      parent: editorEl
    });
  });

  onDestroy(() => view?.destroy());

  // ── Translate ────────────────────────────────────────────────────────────────
  async function translate() {
    if (running) return;
    running = true;
    runError = '';
    result = null;
    try {
      result = await api.translate({
        content: view?.state.doc.toString() ?? editorContent,
        format,
        annotations,
        locators,
        signatureLevel
      });
      activeTab = result.hasErrors ? 'diag' : (format === 'json' ? 'json' : 'xml');
    } catch (e) {
      runError = String(e);
    } finally {
      running = false;
    }
  }

  function clickDiag(d: Diagnostic) {
    if (!view || d.startLine <= 0) return;
    jumpToLocator(view, {
      startLine: d.startLine,
      startChar: d.startChar,
      endLine: d.endLine,
      endChar: d.endChar
    });
  }

  function diagClass(s: string) {
    if (s === 'Error') return 'diag-error';
    if (s === 'Warning') return 'diag-warning';
    return 'diag-info';
  }
</script>

<div class="max-w-5xl mx-auto space-y-4">
  <div class="flex items-baseline gap-4">
    <h1 class="text-xl font-bold text-brand-light">Translate</h1>
    {#if result}
      <span class="text-sm {result.hasErrors ? 'text-red-400' : 'text-green-400'}">
        {result.hasErrors ? `${result.diagnostics.filter(d => d.severity === 'Error').length} error(s)` : `✓ ${result.parsedName}@${result.parsedVersion}`}
      </span>
    {/if}
  </div>

  <!-- Editor -->
  <div class="card p-0 overflow-hidden border border-surface-border rounded-lg">
    <div class="px-3 py-1.5 bg-surface-muted text-xs text-slate-400 border-b border-surface-border">CQL source</div>
    <div bind:this={editorEl}></div>
  </div>

  <!-- Options bar -->
  <div class="card flex flex-wrap items-center gap-4 py-3">
    <label class="flex items-center gap-1.5 text-xs text-slate-400">
      Format
      <select bind:value={format} class="bg-surface-muted text-slate-200 rounded px-2 py-1 text-xs border border-surface-border">
        <option value="both">XML + JSON</option>
        <option value="xml">XML</option>
        <option value="json">JSON</option>
      </select>
    </label>
    <label class="flex items-center gap-1.5 text-xs text-slate-400">
      Signatures
      <select bind:value={signatureLevel} class="bg-surface-muted text-slate-200 rounded px-2 py-1 text-xs border border-surface-border">
        <option value="Overloads">Overloads</option>
        <option value="None">None</option>
        <option value="Differing">Differing</option>
        <option value="All">All</option>
      </select>
    </label>
    <label class="flex items-center gap-1.5 text-xs text-slate-400">
      <input type="checkbox" bind:checked={annotations} class="accent-brand" />
      Annotations
    </label>
    <label class="flex items-center gap-1.5 text-xs text-slate-400">
      <input type="checkbox" bind:checked={locators} class="accent-brand" />
      Locators
    </label>
    <button class="btn-primary ml-auto" onclick={translate} disabled={running}>
      {running ? 'Translating…' : 'Translate'}
    </button>
  </div>

  {#if runError}
    <div class="card border-red-800 text-red-400 text-sm">{runError}</div>
  {/if}

  <!-- Results -->
  {#if result}
    <!-- Tab bar -->
    <div class="flex gap-1 border-b border-surface-border">
      {#each (['xml','json','diag'] as const) as tab}
        {@const label = tab === 'diag' ? `Diagnostics (${result.diagnostics.length})` : tab.toUpperCase()}
        <button
          class="px-4 py-1.5 text-sm rounded-t {activeTab === tab ? 'bg-surface-card text-brand-light border border-b-surface-card border-surface-border' : 'text-slate-500 hover:text-slate-300'}"
          onclick={() => activeTab = tab}
        >{label}</button>
      {/each}
    </div>

    <div class="card mt-0 rounded-tl-none">
      {#if activeTab === 'xml'}
        {#if result.elmXml}
          <pre class="text-xs text-slate-300 overflow-auto max-h-96 whitespace-pre">{result.elmXml}</pre>
        {:else}
          <p class="text-slate-500 text-sm">XML not generated (select format = XML or both).</p>
        {/if}
      {:else if activeTab === 'json'}
        {#if result.elmJson}
          <pre class="text-xs text-slate-300 overflow-auto max-h-96 whitespace-pre">{result.elmJson}</pre>
        {:else}
          <p class="text-slate-500 text-sm">JSON not generated (select format = JSON or both).</p>
        {/if}
      {:else}
        {#if result.diagnostics.length === 0}
          <p class="text-green-400 text-sm">No diagnostics.</p>
        {:else}
          <ul class="space-y-1.5">
            {#each result.diagnostics as d}
              <li class="{diagClass(d.severity)} py-0.5">
                <button
                  type="button"
                  class="text-left w-full"
                  onclick={() => clickDiag(d)}
                  onkeydown={(e) => e.key === 'Enter' && clickDiag(d)}
                  title="Click to jump to source location"
                >
                  <span class="font-semibold uppercase mr-1">{d.severity}</span>
                  {#if d.startLine > 0}
                    <span class="text-slate-500 mr-1">[{d.startLine}:{d.startChar}]</span>
                  {/if}
                  {d.message}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </div>
  {/if}
</div>
