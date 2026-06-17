<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, GarbageData } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	let data = $state<GarbageData | null>(null);
	let error = $state('');
	let loading = $state(true);

	const source    = $derived((widget.config?.source as string) || '');
	const daysAhead = $derived(Number(widget.config?.days_ahead) || 30);
	const maxItems  = $derived(Number(widget.config?.max_items) || 10);
	const showNext  = $derived(!!widget.config?.show_next_only);

	async function fetchData() {
		if (!source) {
			loading = false;
			error = 'Keine iCal-Quelle konfiguriert';
			return;
		}
		loading = true;
		error = '';
		try {
			data = await api.getGarbage(source, daysAhead, maxItems);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Ladefehler';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchData();
		const interval = setInterval(fetchData, 60 * 60 * 1000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		source; daysAhead; maxItems;
		fetchData();
	});

	function formatDate(dateStr: string, daysUntil: number): string {
		const d = new Date(dateStr + 'T00:00:00');
		if (daysUntil === 0) return 'Heute';
		if (daysUntil === 1) return 'Morgen';
		const weekday = d.toLocaleDateString('de-DE', { weekday: 'short' });
		const day = d.toLocaleDateString('de-DE', { day: 'numeric', month: 'short' });
		return `${weekday}, ${day}`;
	}

	function urgencyClass(daysUntil: number): string {
		if (daysUntil === 0) return 'today';
		if (daysUntil === 1) return 'tomorrow';
		if (daysUntil <= 3) return 'soon';
		return '';
	}

	// All events on the same nearest date shown together
	const nextGroup = $derived(
		data?.events?.length
			? data.events.filter(e => e.date === data!.events[0].date)
			: []
	);
	const remaining = $derived(
		data?.events ? data.events.slice(nextGroup.length) : []
	);
</script>

{#if loading}
	<div class="center muted">Lade Kalender…</div>
{:else if error}
	<div class="center error">{error}</div>
{:else if !data || data.events.length === 0}
	<div class="center muted">Keine Termine in den nächsten {daysAhead} Tagen</div>
{:else}
	<div class="garbage">
		{#if nextGroup.length > 0 && !showNext}
			{@const u = urgencyClass(nextGroup[0].days_until)}
			{#each nextGroup as ev (ev.summary)}
				<div class="next-row {u}">
					<span class="next-icon">{ev.icon}</span>
					<div class="next-info">
						<span class="next-summary">{ev.summary}</span>
						<span class="next-date">{formatDate(ev.date, ev.days_until)}</span>
					</div>
					{#if ev.days_until <= 1}
						<span class="badge {u}">
							{ev.days_until === 0 ? 'Heute' : 'Morgen'}
						</span>
					{/if}
				</div>
			{/each}
		{/if}

		{#if !showNext}
			<div class="event-list">
				{#each remaining as ev (ev.date + ev.summary)}
					<div class="event-row {urgencyClass(ev.days_until)}">
						<span class="ev-icon">{ev.icon}</span>
						<span class="ev-summary">{ev.summary}</span>
						<span class="ev-date">{formatDate(ev.date, ev.days_until)}</span>
					</div>
				{/each}
			</div>
		{:else if nextGroup.length > 0}
			<div class="next-only">
				<div class="next-icons-row">
					{#each nextGroup as ev (ev.summary)}
						<span class="next-icon-big">{ev.icon}</span>
					{/each}
				</div>
				{#each nextGroup as ev (ev.summary)}
					<div class="next-label">{ev.summary}</div>
				{/each}
				<div class="next-date-big">{formatDate(nextGroup[0].date, nextGroup[0].days_until)}</div>
			</div>
		{/if}
	</div>
{/if}

<style>
	.center {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		font-size: 13px;
		text-align: center;
	}
	.muted { color: var(--text-muted); }
	.error { color: #f87171; }

	.garbage {
		display: flex;
		flex-direction: column;
		gap: 8px;
		height: 100%;
	}

	.next-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 10px;
		border-radius: 8px;
		background: rgba(255,255,255,0.04);
		border: 1px solid rgba(255,255,255,0.07);
	}

	.next-row.today    { border-color: rgba(251,191,36,0.4); background: rgba(251,191,36,0.07); }
	.next-row.tomorrow { border-color: rgba(96,165,250,0.3); background: rgba(96,165,250,0.05); }

	.next-icon { font-size: 26px; flex-shrink: 0; }

	.next-info {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-width: 0;
	}

	.next-summary {
		font-size: 13px;
		font-weight: 600;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.next-date {
		font-size: 11px;
		color: var(--text-muted);
	}

	.badge {
		font-size: 10px;
		font-weight: 600;
		padding: 2px 7px;
		border-radius: 999px;
		background: rgba(255,255,255,0.1);
		color: var(--text-muted);
		flex-shrink: 0;
	}

	.badge.today    { background: rgba(251,191,36,0.2); color: #fbbf24; }
	.badge.tomorrow { background: rgba(96,165,250,0.2); color: #60a5fa; }

	.event-list {
		display: flex;
		flex-direction: column;
		gap: 3px;
		overflow-y: auto;
		flex: 1;
	}

	.event-row {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 12px;
		padding: 4px 6px;
		border-radius: 6px;
	}

	.event-row.today    { background: rgba(251,191,36,0.07); }
	.event-row.tomorrow { background: rgba(96,165,250,0.05); }
	.event-row.soon     { background: rgba(167,139,250,0.05); }

	.ev-icon { font-size: 16px; flex-shrink: 0; }

	.ev-summary {
		flex: 1;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.ev-date {
		font-size: 11px;
		color: var(--text-muted);
		flex-shrink: 0;
	}

	/* show_next_only mode */
	.next-only {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		flex: 1;
		gap: 6px;
	}

	.next-icons-row {
		display: flex;
		gap: 4px;
	}

	.next-icon-big { font-size: 48px; }
	.next-label    { font-size: 16px; font-weight: 600; color: var(--text); }
	.next-date-big { font-size: 14px; color: var(--text-muted); }
</style>
