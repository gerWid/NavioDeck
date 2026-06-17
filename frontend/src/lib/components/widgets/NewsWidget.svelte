<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, NewsArticle, RssItem, RssSource } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	const cfg         = $derived(widget.config ?? {});
	const mode        = $derived((cfg.mode as string) || 'tagesschau');
	const rssSources  = $derived(Array.isArray(cfg.rss_sources) ? (cfg.rss_sources as RssSource[]) : []);
	const maxItems    = $derived((cfg.max_items as number) || 20);
	const refreshMs   = $derived(((cfg.refresh_interval as number) || 10) * 60_000);

	// Tagesschau params
	const ressort  = $derived((cfg.ressort as string) || '');
	const regions  = $derived(Array.isArray(cfg.regions) ? (cfg.regions as number[]) : []);
	const pageSize = $derived((cfg.page_size as number) || 10);

	let tsArticles = $state<NewsArticle[]>([]);
	let rssItems   = $state<RssItem[]>([]);
	let loading    = $state(true);
	let error      = $state('');
	let lastFetch  = $state<Date | null>(null);

	async function load() {
		loading = true;
		error   = '';
		try {
			if (mode === 'rss') {
				if (rssSources.length === 0) {
					rssItems  = [];
					loading   = false;
					lastFetch = new Date();
					return;
				}
				const data = await api.getRss(rssSources, maxItems);
				rssItems  = data.items ?? [];
			} else {
				const data = await api.getNews({
					regions: regions.length ? regions : undefined,
					ressort: ressort || undefined,
					pageSize,
				});
				tsArticles = (data.news ?? []).slice(0, pageSize);
			}
			lastFetch = new Date();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Fehler beim Laden';
		} finally {
			loading = false;
		}
	}

	function formatTime(isoDate: string): string {
		try {
			const d = new Date(isoDate);
			if (isNaN(d.getTime())) return '';
			const diffMin = Math.floor((Date.now() - d.getTime()) / 60_000);
			if (diffMin < 1)  return 'Gerade eben';
			if (diffMin < 60) return `vor ${diffMin} Min.`;
			const diffH = Math.floor(diffMin / 60);
			if (diffH < 24)   return `vor ${diffH} Std.`;
			if (diffH < 48)   return 'Gestern';
			return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' });
		} catch {
			return '';
		}
	}

	function articleUrl(a: NewsArticle): string {
		if (a.detailsweb) return a.detailsweb;
		if (a.shareURL)   return a.shareURL;
		const s = a.streams;
		if (s) return s.h264m || s.h264xl || s.h264s || '';
		return '';
	}

	function isVideo(a: NewsArticle): boolean {
		return a.type === 'video';
	}

	// Feed/article links are external, untrusted input — strip script URLs.
	function safeUrl(url: string): string {
		const t = (url ?? '').trim().toLowerCase();
		return t.startsWith('javascript:') || t.startsWith('vbscript:') || t.startsWith('data:text/html')
			? '#' : url;
	}

	onMount(() => {
		const id = setInterval(load, refreshMs);
		return () => clearInterval(id);
	});

	$effect(() => {
		void mode;
		void ressort;
		void regions.join(',');
		void pageSize;
		void JSON.stringify(rssSources);
		void maxItems;
		load();
	});
</script>

<div class="news-widget">
	{#if loading && tsArticles.length === 0 && rssItems.length === 0}
		<div class="news-status">Lade Nachrichten…</div>
	{:else if error}
		<div class="news-status error">{error}</div>
	{:else if mode === 'rss'}
		{#if rssSources.length === 0}
			<div class="news-status">Keine Quellen konfiguriert.<br>Bitte im Editor RSS-Feeds hinzufügen.</div>
		{:else if rssItems.length === 0}
			<div class="news-status">Keine Artikel gefunden.</div>
		{:else}
			<ul class="news-list">
				{#each rssItems as item (item.link + item.pub_date)}
					<li class="news-item">
						{#if item.link}
							<a href={safeUrl(item.link)} target="_blank" rel="external noopener noreferrer" class="news-link">
								{#if rssSources.length > 1}
									<span class="source-tag">{item.source_name}</span>
								{/if}
								<span class="title">{item.title}</span>
								{#if item.description}
									<span class="teaser">{item.description}</span>
								{/if}
								<span class="meta">{formatTime(item.pub_date)}</span>
							</a>
						{:else}
							<div class="news-link no-link">
								{#if rssSources.length > 1}
									<span class="source-tag">{item.source_name}</span>
								{/if}
								<span class="title">{item.title}</span>
								{#if item.description}
									<span class="teaser">{item.description}</span>
								{/if}
								<span class="meta">{formatTime(item.pub_date)}</span>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	{:else}
		<!-- Tagesschau mode -->
		{#if tsArticles.length === 0}
			<div class="news-status">Keine Nachrichten gefunden.</div>
		{:else}
			<ul class="news-list">
				{#each tsArticles as a (a.sophoraId)}
					<li class="news-item">
						{#if articleUrl(a)}
							<a href={safeUrl(articleUrl(a))} target="_blank" rel="external noopener noreferrer" class="news-link">
								{#if a.topline}<span class="topline">{a.topline}</span>{/if}
								<span class="title">
									{#if isVideo(a)}<span class="video-badge">▶</span>{/if}{a.title}
								</span>
								{#if a.firstSentence}
									<span class="teaser">{a.firstSentence}</span>
								{:else if a.teaserText}
									<span class="teaser">{a.teaserText}</span>
								{/if}
								<span class="meta">{formatTime(a.date)}</span>
							</a>
						{:else}
							<div class="news-link no-link">
								{#if a.topline}<span class="topline">{a.topline}</span>{/if}
								<span class="title">{a.title}</span>
								{#if a.firstSentence}
									<span class="teaser">{a.firstSentence}</span>
								{:else if a.teaserText}
									<span class="teaser">{a.teaserText}</span>
								{/if}
								<span class="meta">{formatTime(a.date)}</span>
							</div>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	{/if}

	{#if lastFetch && !loading}
		<div class="footer">
			Aktualisiert: {lastFetch.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })}
			<button class="refresh-btn" onclick={load} title="Jetzt aktualisieren">↻</button>
		</div>
	{/if}
</div>

<style>
	.news-widget {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
	}

	.news-status {
		padding: 20px;
		text-align: center;
		color: var(--text-muted);
		font-size: 13px;
		line-height: 1.6;
	}

	.news-status.error { color: #f87171; }

	.news-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 1px;
		overflow-y: auto;
		flex: 1;
		min-height: 0;
	}

	.news-item {
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	}
	.news-item:last-child { border-bottom: none; }

	.news-link {
		display: flex;
		flex-direction: column;
		gap: 3px;
		padding: 9px 10px;
		text-decoration: none;
		color: var(--text);
		border-radius: 6px;
		transition: background 0.12s;
	}

	.news-link:hover { background: rgba(255, 255, 255, 0.07); }
	.news-link.no-link { cursor: default; }
	.news-link.no-link:hover { background: none; }

	.source-tag {
		display: inline-block;
		font-size: 0.65em;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--primary);
		border: 1px solid currentColor;
		padding: 1px 5px 2px;
		border-radius: 4px;
		line-height: 1.4;
		margin-bottom: 1px;
		width: fit-content;
	}

	.topline {
		display: inline-block;
		font-size: var(--topline-font-size, 0.65em);
		font-weight: var(--topline-font-weight, 700);
		font-style: var(--topline-font-style, normal);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--topline-color, var(--primary));
		background: var(--topline-bg, transparent);
		border-width: var(--topline-border-width, 1px);
		border-style: solid;
		border-color: currentColor;
		padding: 1px 5px 2px;
		border-radius: 4px;
		line-height: 1.4;
		margin-bottom: 1px;
	}

	.title {
		font-size: 0.88em;
		font-weight: 600;
		line-height: 1.35;
		color: var(--text);
	}

	.video-badge {
		display: inline-block;
		font-size: 0.7em;
		color: var(--accent, #fb923c);
		margin-right: 4px;
		vertical-align: middle;
	}

	.teaser {
		font-size: 0.78em;
		color: var(--text-muted);
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.meta {
		font-size: 0.72em;
		color: var(--text-muted);
		margin-top: 1px;
	}

	.footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 6px;
		padding: 4px 10px 2px;
		font-size: 0.7em;
		color: var(--text-muted);
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		flex-shrink: 0;
	}

	.refresh-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		font-size: 14px;
		padding: 0 2px;
		line-height: 1;
		transition: color 0.12s;
	}

	.refresh-btn:hover { color: var(--primary); }
</style>
