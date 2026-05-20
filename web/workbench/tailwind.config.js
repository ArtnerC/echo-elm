/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			fontFamily: {
				mono: ['ui-monospace', 'Cascadia Code', 'monospace']
			},
			colors: {
				surface: {
					base: '#0f0f1a',
					card: '#1e1e2e',
					muted: '#2d2d4e',
					border: '#2d2d4e'
				},
				brand: {
					DEFAULT: '#7c3aed',
					light: '#a78bfa',
					muted: '#c4b5fd'
				}
			}
		}
	},
	plugins: []
};
