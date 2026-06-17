<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import ServicesWidget from './widgets/ServicesWidget.svelte';
	import WeatherWidget from './widgets/WeatherWidget.svelte';
	import ClockWidget from './widgets/ClockWidget.svelte';
	import BookmarksWidget from './widgets/BookmarksWidget.svelte';
	import NewsWidget from './widgets/NewsWidget.svelte';
	import DockerWidget from './widgets/DockerWidget.svelte';
	import GarbageWidget from './widgets/GarbageWidget.svelte';
	import FuelWidget from './widgets/FuelWidget.svelte';
	import CalendarWidget from './widgets/CalendarWidget.svelte';

	import type { WidgetStyle } from '$lib/types';

	let { widgetId }: { widgetId: string } = $props();

	// Always derive from the store — stays in sync when config updates via WebSocket or save
	const widget = $derived(dashboard.config?.widgets.find((w) => w.id === widgetId) ?? null);

	function buildCardStyle(s?: WidgetStyle): string {
		const parts: string[] = ['position:relative'];
		if (s?.background_color) parts.push(`background-color:${s.background_color}`);
		if (s?.background_image) {
			parts.push(`background-image:url('${s.background_image}')`);
			parts.push('background-size:cover');
			parts.push('background-position:center');
		}
		if (s?.border_color) parts.push(`border-color:${s.border_color}`);
		if (s?.opacity)       parts.push(`opacity:${s.opacity}`);
		return parts.join(';');
	}

	const cardStyle    = $derived(buildCardStyle(widget?.style));
	const titleStyle   = $derived(widget?.style?.title_color ? `color:${widget.style.title_color}` : '');
	function buildBodyStyle(s?: WidgetStyle): string {
		if (!s) return '';
		const parts: string[] = [];
		if (s.text_color)       { parts.push(`--text:${s.text_color}`, `color:${s.text_color}`); }
		if (s.topline_color)    { parts.push(`--topline-color:${s.topline_color}`); }
		if (s.topline_bg)       { parts.push(`--topline-bg:${s.topline_bg}`); }
		if (s.topline_font_size){ parts.push(`--topline-font-size:${s.topline_font_size}px`); }
		if (s.topline_bold === '0')    { parts.push('--topline-font-weight:400'); }
		if (s.topline_italic === '1')  { parts.push('--topline-font-style:italic'); }
		if (s.topline_border === '0')  { parts.push('--topline-border-width:0'); }
		return parts.join(';');
	}
	const bodyStyle = $derived(buildBodyStyle(widget?.style));

	const components: Record<string, typeof ServicesWidget> = {
		services:  ServicesWidget,
		weather:   WeatherWidget,
		clock:     ClockWidget,
		bookmarks: BookmarksWidget,
		news:      NewsWidget,
		docker:    DockerWidget,
		garbage:   GarbageWidget,
		fuel:      FuelWidget,
		calendar:  CalendarWidget,
	};

	const Component = $derived(widget ? (components[widget.type] ?? ServicesWidget) : null);

	function onEdit() {
		if (widget) dashboard.editingWidget = widget;
	}

	function onDelete() {
		if (widget && confirm(`Widget "${widget.title || widget.type}" löschen?`)) {
			dashboard.deleteWidget(widget.id);
		}
	}
</script>

{#if widget && Component}
	<div class="widget-card" style={cardStyle}>
		<!-- Overlay is always rendered in edit mode; pointer-events:all lets the whole
		     card act as drag target while buttons stop propagation to prevent accidental drag -->
		{#if dashboard.editMode}
			<div class="edit-overlay">
				<div class="drag-hint" title="Verschieben">⠿</div>
				<div class="edit-actions">
					<button
						class="icon-btn"
						onpointerdown={(e) => e.stopPropagation()}
						onclick={onEdit}
						title="Bearbeiten"
					>✏️</button>
					<button
						class="icon-btn danger"
						onpointerdown={(e) => e.stopPropagation()}
						onclick={onDelete}
						title="Löschen"
					>🗑</button>
				</div>
			</div>
		{/if}

		{#if widget.title}
			<div class="widget-header" style={titleStyle}>{widget.title}</div>
		{/if}

		<div class="widget-body" style={bodyStyle}>
			<Component {widget} />
		</div>
	</div>
{/if}

<style>
	.edit-overlay {
		position: absolute;
		inset: 0;
		z-index: 10;
		border-radius: var(--radius);
		border: 2px solid var(--primary);
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		padding: 6px;
		/* pointer-events:all so the overlay absorbs clicks (no accidental link clicks in edit mode)
		   and the whole card becomes a drag surface for GridStack */
		pointer-events: all;
		cursor: grab;
	}

	.edit-overlay:active {
		cursor: grabbing;
	}

	.drag-hint {
		background: var(--primary);
		color: #fff;
		border-radius: 6px;
		width: 28px;
		height: 28px;
		font-size: 16px;
		display: flex;
		align-items: center;
		justify-content: center;
		line-height: 1;
		pointer-events: none;
		user-select: none;
	}

	.edit-actions {
		display: flex;
		gap: 4px;
	}

	.icon-btn {
		background: var(--surface);
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 6px;
		width: 28px;
		height: 28px;
		cursor: pointer;
		font-size: 13px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.icon-btn.danger {
		border-color: rgba(239, 68, 68, 0.4);
	}

	.icon-btn.danger:hover {
		background: rgba(239, 68, 68, 0.25);
	}

	@media (max-width: 768px) {
		.drag-hint { display: none; }
		.edit-overlay { cursor: default; justify-content: flex-end; }
	}
</style>
