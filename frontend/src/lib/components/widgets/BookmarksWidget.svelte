<script lang="ts">
	import type { Widget, WidgetItem } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	const items = $derived(widget.items ?? []);

	const cfg       = $derived(widget.config ?? {});
	const columns   = $derived(Number(cfg.columns) || 1);
	const gap       = $derived(cfg.gap != null ? Number(cfg.gap) : 2);
	const itemAlign = $derived((cfg.item_align as string) || '');

	const GRID_JUSTIFY: Record<string, string> = { left: 'start', center: 'center', right: 'end' };

	function buildGridStyle(cols: number, g: number, cfgVal: Record<string, unknown>, align: string): string {
		let colDef: string;
		if (cols === 0) {
			const fixedCols = Number(cfgVal.grid_cols);
			const minW      = Number(cfgVal.grid_min_w);
			colDef = fixedCols > 0
				? `repeat(${fixedCols}, 1fr)`
				: minW > 0
					? `repeat(auto-fill, minmax(${minW}px, 1fr))`
					: 'repeat(auto-fill, minmax(120px, 1fr))';
		} else if (cols === 1) {
			colDef = '1fr';
		} else {
			colDef = `repeat(${cols},1fr)`;
		}
		const parts = [`display:grid`, `grid-template-columns:${colDef}`, `gap:${g}px`];
		if (GRID_JUSTIFY[align]) parts.push(`justify-items:${GRID_JUSTIFY[align]}`);
		return parts.join(';');
	}
	const gridStyle = $derived(buildGridStyle(columns, gap, cfg, itemAlign));

	const ICON_DEFAULTS: Record<string, number> = { small: 20, medium: 30, large: 44 };

	function iconUrl(icon: string): string {
		if (icon.startsWith('http') || icon.startsWith('/')) return icon;
		return `https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/${icon}.png`;
	}

	function faviconUrl(url: string): string {
		try {
			const origin = new URL(url).origin;
			return `${origin}/favicon.ico`;
		} catch {
			return '';
		}
	}

	function resolveIcon(item: WidgetItem): string {
		if (item.icon && (item.icon.startsWith('http') || !item.icon.includes('.'))) {
			return iconUrl(item.icon);
		}
		return item.icon || faviconUrl(item.url);
	}

	function safeUrl(url: string): string {
		const t = url.trim().toLowerCase();
		return t.startsWith('javascript:') || t.startsWith('vbscript:') || t.startsWith('data:text/html')
			? '#' : url;
	}

	function initials(name: string): string {
		return name.slice(0, 2).toUpperCase();
	}

	function itemStyle(item: WidgetItem): string {
		const parts: string[] = [];
		if (item.item_width)   parts.push(`width:${item.item_width}px`);
		if (item.item_height)  parts.push(`min-height:${item.item_height}px`);
		if (item.font_family)  parts.push(`font-family:'${item.font_family}',var(--font),sans-serif`);
		if (item.bg_color)     parts.push(`background:${item.bg_color}`);
		if (item.border_color) parts.push(`border:1px solid ${item.border_color}`);
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

	function iconBoxStyle(item: WidgetItem, px: number): string {
		const p = [`width:${px}px`, `height:${px}px`, `min-width:${px}px`];
		if (item.icon_bg_color) p.push(`background:${item.icon_bg_color}`);
		return p.join(';');
	}
</script>

<ul class="bookmarks" style={gridStyle}>
	{#each items as item (item.name)}
		{@const size = item.size ?? 'medium'}
		{@const iconPx = ICON_DEFAULTS[size]}
		<li>
			<a
				href={safeUrl(item.url)}
				target={item.target || '_blank'}
				rel="noopener noreferrer"
				class="bookmark size-{size}"
				style={itemStyle(item)}
			>
				<div class="bm-icon" style={iconBoxStyle(item, iconPx)}>
					{#key item.icon}
						<img
							src={resolveIcon(item)}
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
				<div class="bm-text" style={textContainerStyle(item)}>
					<span class="bm-name" style={nameStyle(item)}>{item.name}</span>
					{#if item.description && size !== 'small'}
						<span class="bm-desc" style={descStyle(item)}>{item.description}</span>
					{/if}
				</div>
			</a>
		</li>
	{/each}
</ul>

<style>
	.bookmarks {
		list-style: none;
		/* layout driven by inline style (grid) */
	}

	.bookmark {
		display: flex;
		align-items: center;
		gap: 10px;
		border-radius: 8px;
		text-decoration: none;
		color: var(--text);
		transition: background 0.15s;
		font-size: 13px; /* base; overridden by inline font_size */
		box-sizing: border-box;
	}

	.bookmark:hover { background: rgba(255, 255, 255, 0.07); }

	.size-small  { padding: 5px 8px; }
	.size-medium { padding: 8px 10px; }

	.size-large {
		padding: 12px 12px;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.07);
		border-radius: 10px;
	}

	.size-large:hover { background: rgba(255, 255, 255, 0.08); }

	.bm-icon {
		flex-shrink: 0;
		border-radius: 6px;
		overflow: hidden;
		background: rgba(255, 255, 255, 0.07);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.size-large .bm-icon { border-radius: 8px; }

	.bm-icon img { width: 100%; height: 100%; object-fit: contain; }

	.fallback {
		font-size: 0.75em;
		font-weight: 700;
		color: var(--primary);
	}

	.hidden { display: none !important; }

	.bm-text {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-width: 0;
	}

	.bm-name {
		font-size: 1em;
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		width: 100%;
		text-align: var(--item-text-align, left);
	}

	.size-small  .bm-name { font-size: 0.9em; }
	.size-large  .bm-name { font-size: 1.05em; font-weight: 600; }

	.bm-desc {
		font-size: 0.85em;
		color: var(--text-muted);
		overflow: hidden;
		text-overflow: ellipsis;
		width: 100%;
		text-align: var(--item-text-align, left);
	}

	.size-large .bm-desc { white-space: normal; }
</style>
