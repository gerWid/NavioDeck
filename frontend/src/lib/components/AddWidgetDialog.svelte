<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import type { Widget, WidgetType } from '$lib/types';
	import { WIDGET_TEMPLATES, DEFAULT_SIZES } from '$lib/types';

	function close() {
		dashboard.addWidgetOpen = false;
	}

	const widgetTypes: { type: WidgetType; label: string; icon: string; desc: string }[] = [
		{ type: 'services', label: 'Services', icon: '🚀', desc: 'Apps und Dienste mit Icons und Links' },
		{ type: 'weather', label: 'Wetter', icon: '⛅', desc: 'Aktuelles Wetter und Vorhersage via Open-Meteo' },
		{ type: 'clock', label: 'Uhr', icon: '🕐', desc: 'Uhrzeit und Datum' },
		{ type: 'bookmarks', label: 'Lesezeichen', icon: '🔖', desc: 'Linkliste mit Beschreibungen' },
		{ type: 'news', label: 'Nachrichten', icon: '📰', desc: 'Tagesschau-Newsticker nach Thema und Region' },
		{ type: 'docker', label: 'Docker', icon: '🐳', desc: 'Container-Status via Docker API oder Socket' },
		{ type: 'garbage', label: 'Müllkalender', icon: '🗑️', desc: 'Nächste Abholtermine aus einer iCal-Datei' },
		{ type: 'fuel', label: 'Benzinpreise', icon: '⛽', desc: 'Aktuelle Spritpreise via Tankerkönig-API' },
		{ type: 'calendar', label: 'Kalender', icon: '📅', desc: 'Termine aus Google Kalender, iCal oder .ics-Datei' },
	];

	function generateId(type: string): string {
		return `${type}-${Date.now().toString(36)}`;
	}

	async function addWidget(type: WidgetType) {
		const template = WIDGET_TEMPLATES[type];
		const size = DEFAULT_SIZES[type];
		const widget: Widget = {
			...template,
			id: generateId(type),
			position: { x: 0, y: 0, ...size },
		};
		await dashboard.addWidget(widget);
		dashboard.editingWidget = widget;
		close();
	}
</script>

<div class="overlay" onclick={close} onkeydown={(e) => { if (e.key === 'Escape') close(); }} role="dialog" aria-modal="true" tabindex="-1">
	<div class="dialog" onclick={(e) => e.stopPropagation()} role="presentation">
		<div class="dialog-header">
			<span>Widget hinzufügen</span>
			<button class="close-btn" onclick={close}>✕</button>
		</div>

		<div class="widget-types">
			{#each widgetTypes as wt}
				<button class="wt-card" onclick={() => addWidget(wt.type)}>
					<span class="wt-icon">{wt.icon}</span>
					<span class="wt-label">{wt.label}</span>
					<span class="wt-desc">{wt.desc}</span>
				</button>
			{/each}
		</div>
	</div>
</div>

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0,0,0,0.6);
		backdrop-filter: blur(4px);
		z-index: 60;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.dialog {
		background: var(--surface);
		border: 1px solid var(--border-subtle);
		border-radius: 16px;
		width: 500px;
		max-width: 95vw;
		overflow: hidden;
	}

	.dialog-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 18px 22px;
		border-bottom: 1px solid var(--border-subtle);
		font-weight: 600;
		font-size: 15px;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 18px;
		cursor: pointer;
	}

	.widget-types {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
		padding: 20px;
	}

	.wt-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 20px 14px;
		background: rgba(255,255,255,0.04);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 12px;
		cursor: pointer;
		text-align: center;
		color: var(--text);
		font-family: var(--font);
		transition: all 0.15s;
	}

	.wt-card:hover {
		background: rgba(129, 140, 248, 0.1);
		border-color: var(--primary);
		transform: translateY(-2px);
	}

	.wt-icon { font-size: 32px; }
	.wt-label { font-size: 14px; font-weight: 600; }
	.wt-desc { font-size: 11px; color: var(--text-muted); line-height: 1.4; }
</style>
