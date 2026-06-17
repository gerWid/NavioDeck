<script lang="ts">
	import type { Widget, WidgetItem } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	const items   = $derived(widget.items ?? []);
	const cfg     = $derived(widget.config ?? {});
	const columns   = $derived(cfg.columns != null ? Number(cfg.columns) : null);
	const gap       = $derived(cfg.gap != null ? Number(cfg.gap) : 8);
	const itemAlign = $derived((cfg.item_align as string) || '');

	const inGridMode = $derived(columns !== null);

	const FLEX_JUSTIFY: Record<string, string> = { left: 'flex-start', center: 'center', right: 'flex-end' };
	const GRID_JUSTIFY: Record<string, string> = { left: 'start',      center: 'center', right: 'end' };

	function buildContainerStyle(cols: number | null, g: number, cfgVal: Record<string, unknown>, align: string): string {
		const parts: string[] = [`gap:${g}px`];
		if (cols !== null) {
			let colDef: string;
			if (cols > 0) {
				colDef = `repeat(${cols},1fr)`;
			} else {
				const fixedCols = Number(cfgVal.grid_cols);
				const minW      = Number(cfgVal.grid_min_w);
				colDef = fixedCols > 0
					? `repeat(${fixedCols}, 1fr)`
					: minW > 0
						? `repeat(auto-fill, minmax(${minW}px, 1fr))`
						: 'repeat(auto-fill, minmax(100px, 1fr))';
			}
			parts.push('display:grid', `grid-template-columns:${colDef}`);
			if (GRID_JUSTIFY[align]) parts.push(`justify-items:${GRID_JUSTIFY[align]}`);
		} else {
			if (FLEX_JUSTIFY[align]) parts.push(`justify-content:${FLEX_JUSTIFY[align]}`);
		}
		return parts.join(';');
	}

	const containerStyle = $derived(buildContainerStyle(columns, gap, cfg, itemAlign));

	const ICON_DEFAULTS: Record<string, number> = { small: 32, medium: 44, large: 68 };

	function iconUrl(icon: string): string {
		if (icon.startsWith('http') || icon.startsWith('/')) return icon;
		return `https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/${icon}.png`;
	}

	function initials(name: string): string {
		return name.slice(0, 2).toUpperCase();
	}

	function safeUrl(url: string): string {
		const t = url.trim().toLowerCase();
		return t.startsWith('javascript:') || t.startsWith('vbscript:') || t.startsWith('data:text/html')
			? '#' : url;
	}

	function openItem(item: WidgetItem) {
		const url = safeUrl(item.url);
		if (url !== '#') window.open(url, item.target || '_blank');
	}

	function itemStyle(item: WidgetItem): string {
		const parts: string[] = [];
		if (item.item_width)   parts.push(`width:${item.item_width}px`, `flex:0 0 ${item.item_width}px`, `max-width:${item.item_width}px`);
		if (item.item_height)  parts.push(`height:${item.item_height}px`, `overflow:hidden`);
		if (item.font_family)  parts.push(`font-family:'${item.font_family}',var(--font),sans-serif`);
		if (item.bg_color)     parts.push(`background:${item.bg_color}`);
		if (item.border_color) parts.push(`border-color:${item.border_color}`);
		return parts.join(';');
	}

	function nameStyle(item: WidgetItem): string {
		const p: string[] = [];
		if (item.name_font_size) p.push(`font-size:${item.name_font_size}px`);
		if (item.text_color)     p.push(`color:${item.text_color}`);
		return p.join(';');
	}

	function descStyle(item: WidgetItem): string {
		const p: string[] = [];
		if (item.desc_font_size) p.push(`font-size:${item.desc_font_size}px`);
		if (item.desc_color)     p.push(`color:${item.desc_color}`);
		return p.join(';');
	}

	function textContainerStyle(item: WidgetItem): string {
		if (!item.text_align || item.text_align === 'left') return '';
		return `--item-text-align:${item.text_align}`;
	}

	function iconStyle(item: WidgetItem, px: number): string {
		const p = [`width:${px}px`, `height:${px}px`];
		if (item.icon_bg_color) p.push(`background:${item.icon_bg_color}`);
		return p.join(';');
	}
</script>

<div class="services-grid" class:grid-mode={inGridMode} style={containerStyle}>
	{#each items as item (item.name)}
		{@const size = item.size ?? 'medium'}
		{@const iconPx = item.icon_size ?? ICON_DEFAULTS[size]}
		<button
			class="service-item size-{size}"
			style={itemStyle(item)}
			onclick={() => openItem(item)}
			title={item.description || item.name}
		>
			<div class="service-icon" style={iconStyle(item, iconPx)}>
				{#key item.icon}
					<img
						src={iconUrl(item.icon)}
						alt={item.name}
						loading="lazy"
						onerror={(e) => {
							const img = e.currentTarget as HTMLImageElement;
							img.style.display = 'none';
							img.nextElementSibling?.classList.remove('hidden');
						}}
					/>
					<span class="fallback hidden">{initials(item.name)}</span>
				{/key}
			</div>
			{#if size !== 'small'}
				<div class="service-text" style={textContainerStyle(item)}>
					<span class="service-name" style={nameStyle(item)}>{item.name}</span>
					{#if size === 'large' && item.description}
						<span class="service-desc" style={descStyle(item)}>{item.description}</span>
					{/if}
				</div>
			{/if}
		</button>
	{/each}
</div>

<style>
	.services-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding: 4px 0;
		align-content: flex-start;
	}

	.service-item {
		display: flex;
		align-items: center;
		gap: 8px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.07);
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.15s;
		color: var(--text);
		font-family: var(--font);
		font-size: 13px; /* base; overridden by inline font_size */
		box-sizing: border-box;
	}

	.service-item:hover {
		background: rgba(255, 255, 255, 0.1);
		border-color: var(--primary);
		transform: translateY(-2px);
	}

	.size-small {
		flex-direction: column;
		justify-content: center;
		padding: 8px;
		flex: 0 0 auto;
	}

	.size-medium {
		flex-direction: column;
		justify-content: center;
		padding: 10px 8px;
		text-align: center;
		flex: 1 1 80px;
		max-width: 140px;
	}

	.size-large {
		flex-direction: row;
		padding: 12px 16px;
		gap: 14px;
		flex: 1 1 100%;
	}

	.service-icon {
		border-radius: 8px;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(255, 255, 255, 0.08);
		flex-shrink: 0;
	}

	.service-icon img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.fallback {
		font-size: 0.85em;
		font-weight: 700;
		color: var(--primary);
	}

	.hidden { display: none !important; }

	.service-text {
		display: flex;
		flex-direction: column;
		flex: 1;
		gap: 3px;
		min-width: 0;
	}

	.service-name {
		font-size: 0.85em;
		font-weight: 500;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		width: 100%;
		text-align: var(--item-text-align, center);
	}

	.size-large .service-name {
		font-size: 1em;
		font-weight: 600;
		white-space: normal;
		text-align: var(--item-text-align, left);
	}

	.service-desc {
		font-size: 0.85em;
		color: var(--text-muted);
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		width: 100%;
		text-align: var(--item-text-align, left);
	}

	/* In grid mode: remove flex sizing constraints so grid controls layout */
	.grid-mode .service-item {
		max-width: none;
		flex: unset;
		box-sizing: border-box;
	}
</style>
