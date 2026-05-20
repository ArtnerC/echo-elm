<script lang="ts">
  import '../app.css';
  import { page } from '$app/stores';

  const links = [
    { href: '/', label: 'Libraries' },
    { href: '/translate', label: 'Translate' },
  ];
</script>

<div class="min-h-screen flex flex-col">
  <header class="border-b border-surface-border bg-surface-card px-6 py-3 flex items-center gap-6">
    <span class="font-bold text-brand-light tracking-tight">echo-elm</span>
    <span class="text-slate-500 text-xs">workbench</span>
    <nav class="flex gap-4 ml-4">
      {#each links as l}
        <a
          href={l.href}
          class="text-sm {$page.url.pathname === l.href ? 'text-brand-light' : 'text-slate-400 hover:text-slate-200'}"
        >{l.label}</a>
      {/each}
    </nav>
  </header>

  <main class="flex-1 p-6">
    <slot />
  </main>

  <footer class="border-t border-surface-border text-center text-slate-600 text-xs py-2">
    echo-elm · CQL 1.5.3 → ELM R1
  </footer>
</div>
