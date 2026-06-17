<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Wallpaper } from '$lib/types';

	let {
		value = '',
		onchange,
	}: {
		value: string;
		onchange: (url: string) => void;
	} = $props();

	let wallpapers = $state<Wallpaper[]>([]);
	let uploading = $state(false);
	let fileInput: HTMLInputElement;

	onMount(async () => {
		try {
			wallpapers = await api.getWallpapers();
		} catch {}
	});

	async function handleUpload(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;
		uploading = true;
		try {
			const w = await api.uploadWallpaper(file);
			wallpapers = [...wallpapers, w];
			onchange(w.url);
		} catch (err) {
			console.error('Upload failed:', err);
		} finally {
			uploading = false;
			fileInput.value = '';
		}
	}

	async function remove(w: Wallpaper, e: MouseEvent) {
		e.stopPropagation();
		if (!confirm(`"${w.name}" löschen?`)) return;
		try {
			await api.deleteWallpaper(w.name);
			wallpapers = wallpapers.filter((wp) => wp.name !== w.name);
			if (value === w.url) onchange('');
		} catch (err) {
			console.error('Delete failed:', err);
		}
	}
</script>

<div class="wallpaper-picker">
	<div class="thumbs">
		<!-- "None" option -->
		<button
			class="thumb none-thumb"
			class:selected={!value}
			onclick={() => onchange('')}
			title="Kein Hintergrundbild"
		>
			<span>–</span>
		</button>

		{#each wallpapers as w (w.name)}
			<div
				class="thumb img-thumb"
				class:selected={value === w.url}
				onclick={() => onchange(w.url)}
				role="button"
				tabindex="0"
				title={w.name}
				onkeydown={(e) => e.key === 'Enter' && onchange(w.url)}
			>
				<img src={w.url} alt={w.name} />
				<button
					class="del-btn"
					onclick={(e) => remove(w, e)}
					title="Löschen"
					onpointerdown={(e) => e.stopPropagation()}
				>✕</button>
			</div>
		{/each}

		<!-- Upload button -->
		<button
			class="thumb upload-thumb"
			onclick={() => fileInput.click()}
			title="Bild hochladen"
			disabled={uploading}
		>
			{#if uploading}
				<span class="spin">⟳</span>
			{:else}
				<span>+</span>
			{/if}
		</button>
	</div>

	<input
		bind:this={fileInput}
		type="file"
		accept=".jpg,.jpeg,.png,.gif,.webp,.avif"
		style="display:none"
		onchange={handleUpload}
	/>
</div>

<style>
	.wallpaper-picker {
		width: 100%;
	}

	.thumbs {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.thumb {
		width: 72px;
		height: 48px;
		border-radius: 6px;
		border: 2px solid rgba(255, 255, 255, 0.1);
		overflow: hidden;
		cursor: pointer;
		position: relative;
		flex-shrink: 0;
		transition: border-color 0.15s;
	}

	.thumb.selected {
		border-color: var(--primary);
	}

	.none-thumb {
		background: rgba(255, 255, 255, 0.05);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 20px;
		color: var(--text-muted);
	}

	.none-thumb:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.img-thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
		pointer-events: none;
	}

	.del-btn {
		position: absolute;
		top: 2px;
		right: 2px;
		width: 16px;
		height: 16px;
		background: rgba(0, 0, 0, 0.7);
		border: none;
		border-radius: 3px;
		color: #fff;
		font-size: 9px;
		cursor: pointer;
		display: none;
		align-items: center;
		justify-content: center;
		padding: 0;
		line-height: 1;
	}

	.img-thumb:hover .del-btn {
		display: flex;
	}

	.del-btn:hover {
		background: rgba(239, 68, 68, 0.85);
	}

	.upload-thumb {
		background: rgba(255, 255, 255, 0.05);
		border-style: dashed;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 22px;
		color: var(--text-muted);
	}

	.upload-thumb:hover:not(:disabled) {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(255, 255, 255, 0.25);
	}

	.upload-thumb:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.spin {
		display: inline-block;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
