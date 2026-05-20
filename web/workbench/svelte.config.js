import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			// Output directly into the Go embed target so `go build` picks it up
			// without a copy step.
			pages: '../../internal/ui/static',
			assets: '../../internal/ui/static',
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		prerender: {
			handleHttpError: 'warn'
		}
	}
};

export default config;
