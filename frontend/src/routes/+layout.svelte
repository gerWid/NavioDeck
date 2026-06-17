<script lang="ts">
	import '../app.css';
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();

	const theme = $derived(dashboard.config?.theme);

	function applyTheme() {
		const mode = dashboard.colorMode;
		if (!theme) return;
		const r = document.documentElement;

		if (mode === 'light') {
			r.style.setProperty('--bg-color', '#f0f2f5');
			r.style.setProperty('--card-color', 'rgba(255,255,255,0.95)');
			r.style.setProperty('--text', '#1a1a1a');
			r.style.setProperty('--text-muted', 'rgba(26,26,26,0.5)');
			r.style.setProperty('--surface', '#ffffff');
			r.style.setProperty('--surface-raised', '#f4f5f7');
			r.style.setProperty('--bar-bg', 'rgba(240,242,245,0.88)');
			r.style.setProperty('--bar-bg-active', 'rgba(240,242,245,0.97)');
			r.style.setProperty('--bar-border', 'rgba(0,0,0,0.08)');
			r.style.setProperty('--item-hover', 'rgba(0,0,0,0.04)');
			r.style.setProperty('--border-subtle', 'rgba(0,0,0,0.08)');
		} else if (mode === 'custom') {
			// User's configured theme colours from dashboard.yaml
			r.style.setProperty('--bg-color', theme.background_color || '#111111');
			r.style.setProperty('--card-color', theme.card_color || 'rgba(28,28,28,0.97)');
			r.style.setProperty('--text', theme.text_color || '#e8eaed');
			r.style.setProperty('--text-muted', 'rgba(241,245,249,0.55)');
			r.style.setProperty('--surface', theme.card_color || 'rgba(28,28,28,0.97)');
			r.style.setProperty('--surface-raised', theme.background_color || '#111111');
			r.style.setProperty('--bar-bg', 'rgba(0,0,0,0.35)');
			r.style.setProperty('--bar-bg-active', 'rgba(0,0,0,0.55)');
			r.style.setProperty('--bar-border', 'rgba(255,255,255,0.06)');
			r.style.setProperty('--item-hover', 'rgba(255,255,255,0.05)');
			r.style.setProperty('--border-subtle', 'rgba(255,255,255,0.08)');
		} else {
			// True dark: black/charcoal tones
			r.style.setProperty('--bg-color', '#111111');
			r.style.setProperty('--card-color', 'rgba(28,28,28,0.97)');
			r.style.setProperty('--text', '#e8eaed');
			r.style.setProperty('--text-muted', 'rgba(232,234,237,0.5)');
			r.style.setProperty('--surface', '#1c1c1c');
			r.style.setProperty('--surface-raised', '#252525');
			r.style.setProperty('--bar-bg', 'rgba(17,17,17,0.82)');
			r.style.setProperty('--bar-bg-active', 'rgba(17,17,17,0.96)');
			r.style.setProperty('--bar-border', 'rgba(255,255,255,0.06)');
			r.style.setProperty('--item-hover', 'rgba(255,255,255,0.05)');
			r.style.setProperty('--border-subtle', 'rgba(255,255,255,0.08)');
		}

		r.style.setProperty('--primary', theme.primary_color || '#818cf8');
		r.style.setProperty('--accent', theme.accent_color || '#fb923c');
		r.style.setProperty('--radius', `${theme.border_radius ?? 14}px`);
		r.style.setProperty('--font', `'${theme.font || 'Inter'}', system-ui, sans-serif`);

		r.setAttribute('data-theme', mode);

		const body = document.body;
		if (theme.background_image && mode === 'custom') {
			body.style.backgroundImage = `url('${theme.background_image}')`;
			body.style.backgroundSize = 'cover';
			body.style.backgroundPosition = 'center';
			body.style.backgroundAttachment = 'fixed';
		} else {
			body.style.backgroundImage = '';
		}
	}

	$effect(() => {
		applyTheme();
	});

	onMount(() => {
		const saved = localStorage.getItem('color-mode') as 'dark' | 'light' | 'custom' | null;
		if (saved === 'light' || saved === 'dark' || saved === 'custom') {
			dashboard.colorMode = saved;
		}
		dashboard.load();
	});
</script>

<div class:glass={theme?.glass_effect}>
	{@render children()}
</div>
