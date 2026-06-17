<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';

	const modeOrder: ('dark' | 'light' | 'custom')[] = ['dark', 'light', 'custom'];
	const modeIcon: Record<string, string>  = { dark: '🌙', light: '☀', custom: '🎨' };
	const modeTitle: Record<string, string> = {
		dark:   'Dunkelmodus → Hellmodus wechseln',
		light:  'Hellmodus → Eigene Farben wechseln',
		custom: 'Eigene Farben → Dunkelmodus wechseln',
	};

	function cycleMode() {
		const idx = modeOrder.indexOf(dashboard.colorMode);
		dashboard.colorMode = modeOrder[(idx + 1) % modeOrder.length];
		localStorage.setItem('color-mode', dashboard.colorMode);
	}
</script>

<header class="edit-bar" class:active={dashboard.editMode}>
	<div class="left">
		<span class="logo">🏠 NavioDeck</span>
	</div>

	<div class="right">
		{#if dashboard.editMode}
			<button class="btn btn-ghost" onclick={() => (dashboard.addWidgetOpen = true)}>
				+ Widget
			</button>
			<button class="btn btn-ghost" onclick={() => (dashboard.settingsOpen = true)}>
				⚙ Design
			</button>
			<button class="btn btn-primary" onclick={() => (dashboard.editMode = false)}>
				✓ Fertig
			</button>
		{:else}
			<button class="btn btn-ghost" onclick={() => (dashboard.editMode = true)}>
				✏ Bearbeiten
			</button>
		{/if}
		<button class="btn btn-ghost mode-toggle" onclick={cycleMode}
			title={modeTitle[dashboard.colorMode]}>
			{modeIcon[dashboard.colorMode]}
		</button>
	</div>
</header>

<style>
	.edit-bar {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 40;
		height: 52px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 20px;
		background: var(--bar-bg, rgba(17, 17, 17, 0.82));
		backdrop-filter: blur(12px);
		border-bottom: 1px solid var(--bar-border, rgba(255, 255, 255, 0.06));
		transition: border-color 0.2s, background 0.2s;
	}

	.edit-bar.active {
		border-bottom-color: rgba(129, 140, 248, 0.4);
		background: var(--bar-bg-active, rgba(17, 17, 17, 0.96));
	}

	.mode-toggle {
		padding: 6px 10px;
		font-size: 15px;
	}

	.logo {
		font-size: 16px;
		font-weight: 700;
		color: var(--text);
		opacity: 0.9;
	}

	.right {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.btn {
		padding: 6px 14px;
		font-size: 13px;
	}

	@media (max-width: 480px) {
		.edit-bar { padding: 0 12px; }
		.logo { font-size: 14px; }
		.right { gap: 5px; }
		.btn { padding: 5px 10px; font-size: 12px; }
	}
</style>
