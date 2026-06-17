<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Icon } from '$lib/types';

	let {
		value = '',
		onchange,
	}: {
		value: string;
		onchange: (url: string) => void;
	} = $props();

	let icons = $state<Icon[]>([]);
	let uploading = $state(false);
	let fileInput: HTMLInputElement;

	onMount(async () => {
		try {
			icons = await api.getIcons();
		} catch {}
	});

	async function handleUpload(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;
		uploading = true;
		try {
			const icon = await api.uploadIcon(file);
			icons = [...icons, icon];
			onchange(icon.url);
		} catch (err) {
			console.error('Upload failed:', err);
		} finally {
			uploading = false;
			fileInput.value = '';
		}
	}

	async function remove(icon: Icon, e: MouseEvent) {
		e.stopPropagation();
		if (!confirm(`"${icon.name}" löschen?`)) return;
		try {
			await api.deleteIcon(icon.name);
			icons = icons.filter((i) => i.name !== icon.name);
			if (value === icon.url) onchange('');
		} catch (err) {
			console.error('Delete failed:', err);
		}
	}
</script>

{#if icons.length > 0 || true}
	<div class="icon-picker">
		{#each icons as icon (icon.name)}
			<div
				class="icon-thumb"
				class:selected={value === icon.url}
				onclick={() => onchange(icon.url)}
				role="button"
				tabindex="0"
				title={icon.name}
				onkeydown={(e) => e.key === 'Enter' && onchange(icon.url)}
			>
				<img src={icon.url} alt={icon.name} />
				<button
					class="del-btn"
					onclick={(e) => remove(icon, e)}
					onpointerdown={(e) => e.stopPropagation()}
					title="Löschen"
				>✕</button>
			</div>
		{/each}

		<button
			class="icon-thumb upload-thumb"
			onclick={() => fileInput.click()}
			title="Icon hochladen"
			disabled={uploading}
		>
			{#if uploading}
				<span class="spin">⟳</span>
			{:else}
				<span>+</span>
			{/if}
		</button>
	</div>
{/if}

<input
	bind:this={fileInput}
	type="file"
	accept=".jpg,.jpeg,.png,.gif,.webp,.avif,.svg"
	style="display:none"
	onchange={handleUpload}
/>

<style>
	.icon-picker {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-top: 6px;
	}

	.icon-thumb {
		width: 44px;
		height: 44px;
		border-radius: 8px;
		border: 2px solid rgba(255, 255, 255, 0.1);
		overflow: hidden;
		cursor: pointer;
		position: relative;
		flex-shrink: 0;
		background: rgba(255, 255, 255, 0.05);
		transition: border-color 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.icon-thumb.selected {
		border-color: var(--primary);
	}

	.icon-thumb img {
		width: 32px;
		height: 32px;
		object-fit: contain;
		pointer-events: none;
	}

	.del-btn {
		position: absolute;
		top: 1px;
		right: 1px;
		width: 14px;
		height: 14px;
		background: rgba(0, 0, 0, 0.75);
		border: none;
		border-radius: 3px;
		color: #fff;
		font-size: 8px;
		cursor: pointer;
		display: none;
		align-items: center;
		justify-content: center;
		padding: 0;
		line-height: 1;
	}

	.icon-thumb:hover .del-btn {
		display: flex;
	}

	.del-btn:hover {
		background: rgba(239, 68, 68, 0.85);
	}

	.upload-thumb {
		border-style: dashed;
		font-size: 20px;
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
