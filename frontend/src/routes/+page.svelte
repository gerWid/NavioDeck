<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import Grid from '$lib/components/Grid.svelte';
	import EditBar from '$lib/components/EditBar.svelte';
	import SettingsPanel from '$lib/components/SettingsPanel.svelte';
	import AddWidgetDialog from '$lib/components/AddWidgetDialog.svelte';
	import WidgetEditor from '$lib/components/WidgetEditor.svelte';
	import LoginOverlay from '$lib/components/LoginOverlay.svelte';

	const widgets = $derived(dashboard.config?.widgets ?? []);
</script>

{#if dashboard.authenticated === false}
	<LoginOverlay />
{:else}
	<EditBar />

	<main class="dashboard" class:edit-mode={dashboard.editMode}>
		{#if dashboard.config}
			<Grid {widgets} />
		{:else}
			<div class="loading">
				<span>Lade Dashboard…</span>
			</div>
		{/if}
	</main>

	{#if dashboard.settingsOpen}
		<SettingsPanel />
	{/if}

	{#if dashboard.addWidgetOpen}
		<AddWidgetDialog />
	{/if}

	{#if dashboard.editingWidget}
		<WidgetEditor />
	{/if}
{/if}

<style>
	.dashboard {
		padding-top: 60px;
		min-height: 100vh;
	}

	@media (max-width: 768px) {
		.dashboard {
			padding-top: 52px;
			padding-bottom: 16px;
		}
	}

	.loading {
		display: flex;
		align-items: center;
		justify-content: center;
		height: calc(100vh - 60px);
		font-size: 14px;
		color: var(--text-muted);
	}
</style>
