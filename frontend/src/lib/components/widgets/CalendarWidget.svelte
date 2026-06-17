<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, CalendarEvent, CalendarSource } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	type RichEvent = CalendarEvent & { calName: string; calColor: string };

	const DEFAULT_COLORS = ['#818cf8','#fb923c','#34d399','#f472b6','#60a5fa','#a78bfa','#fbbf24'];

	let allEvents = $state<RichEvent[]>([]);
	let error     = $state('');
	let loading   = $state(true);

	const daysAhead = $derived(Number(widget.config?.days_ahead) || 30);
	const maxItems  = $derived(Number(widget.config?.max_items)  || 20);

	// Serialize sources so $effect can track changes to array contents
	const sourcesKey = $derived(JSON.stringify(widget.config?.sources ?? widget.config?.source ?? ''));

	function getSources(): { name: string; url: string; color: string }[] {
		const cfg = widget.config ?? {};
		if (Array.isArray(cfg.sources) && cfg.sources.length > 0) {
			return (cfg.sources as CalendarSource[])
				.filter(s => s.url)
				.map((s, i) => ({
					name:  s.name  || `Kalender ${i + 1}`,
					url:   s.url,
					color: s.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length],
				}));
		}
		// backward-compat: old single-source field
		if (cfg.source) {
			return [{ name: 'Kalender', url: cfg.source as string, color: DEFAULT_COLORS[0] }];
		}
		return [];
	}

	async function fetchAll() {
		const srcs = getSources();
		if (!srcs.length) {
			loading = false;
			error = 'Keine iCal-Quelle konfiguriert';
			return;
		}
		loading = true;
		error = '';
		try {
			const results = await Promise.allSettled(
				srcs.map(s =>
					api.getCalendar(s.url, daysAhead, 500).then(d =>
						(d.events ?? []).map(ev => ({ ...ev, calName: s.name, calColor: s.color } as RichEvent))
					)
				)
			);

			const merged: RichEvent[] = [];
			let anyErr = false;
			for (const r of results) {
				if (r.status === 'fulfilled') merged.push(...r.value);
				else anyErr = true;
			}

			merged.sort((a, b) => {
				const d = a.date.localeCompare(b.date);
				return d !== 0 ? d : (a.time || '').localeCompare(b.time || '');
			});
			allEvents = merged.slice(0, maxItems);
			if (!merged.length && anyErr) error = 'Fehler beim Laden eines Kalenders';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Ladefehler';
		} finally {
			loading = false;
		}
	}

	// Periodic refresh — initial fetch is handled by the $effect below
	onMount(() => {
		const iv = setInterval(fetchAll, 30 * 60 * 1000);
		return () => clearInterval(iv);
	});

	$effect(() => {
		sourcesKey; daysAhead; maxItems;
		fetchAll();
	});

	// Only show calendar name badge when multiple sources are configured
	const multiCal = $derived(getSources().length > 1);

	const groups = $derived(() => {
		if (!allEvents.length) return [];
		const result: { date: string; daysUntil: number; events: RichEvent[] }[] = [];
		for (const ev of allEvents) {
			const last = result[result.length - 1];
			if (last && last.date === ev.date) last.events.push(ev);
			else result.push({ date: ev.date, daysUntil: ev.days_until, events: [ev] });
		}
		return result;
	});

	function formatDayLabel(dateStr: string, daysUntil: number): string {
		if (daysUntil === 0) return 'Heute';
		if (daysUntil === 1) return 'Morgen';
		const d = new Date(dateStr + 'T00:00:00');
		return d.toLocaleDateString('de-DE', { weekday: 'short', day: 'numeric', month: 'short' });
	}

	function urgencyClass(daysUntil: number): string {
		if (daysUntil === 0) return 'today';
		if (daysUntil === 1) return 'tomorrow';
		if (daysUntil <= 3) return 'soon';
		return '';
	}

	function formatTimeRange(ev: RichEvent): string {
		if (ev.all_day) return 'Ganztag';
		if (!ev.time) return '';
		if (ev.end_time && ev.end_date === ev.date) return `${ev.time}–${ev.end_time}`;
		return ev.time;
	}
</script>

{#if loading}
	<div class="center muted">Lade Kalender…</div>
{:else if error}
	<div class="center error">{error}</div>
{:else if !allEvents.length}
	<div class="center muted">Keine Termine in den nächsten {daysAhead} Tagen</div>
{:else}
	<div class="calendar">
		{#each groups() as group (group.date)}
			{@const uc = urgencyClass(group.daysUntil)}
			<div class="day-group {uc}">
				<div class="day-header {uc}">
					<span class="day-label">{formatDayLabel(group.date, group.daysUntil)}</span>
					{#if uc === 'today' || uc === 'tomorrow'}
						<span class="day-badge {uc}">{uc === 'today' ? 'Heute' : 'Morgen'}</span>
					{/if}
				</div>
				{#each group.events as ev (ev.summary + ev.time + ev.calName)}
					<div class="event {uc}">
						<div class="event-time" class:allday={ev.all_day}>
							{formatTimeRange(ev)}
						</div>
						<div class="event-content">
							<div class="event-title-row">
								<span class="cal-dot" style="background:{ev.calColor}"></span>
								<span class="event-summary">{ev.summary}</span>
							</div>
							{#if multiCal}
								<span class="event-cal-name" style="color:{ev.calColor}">{ev.calName}</span>
							{/if}
							{#if ev.location}
								<span class="event-location">📍 {ev.location}</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/each}
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

	.calendar {
		display: flex;
		flex-direction: column;
		gap: 8px;
		height: 100%;
		overflow-y: auto;
	}

	.day-group {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.day-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 4px 8px 3px;
		border-radius: 6px 6px 0 0;
		background: rgba(255,255,255,0.04);
		border-left: 3px solid rgba(255,255,255,0.15);
	}
	.day-header.today    { border-left-color: #fbbf24; background: rgba(251,191,36,0.08); }
	.day-header.tomorrow { border-left-color: #60a5fa; background: rgba(96,165,250,0.07); }
	.day-header.soon     { border-left-color: rgba(167,139,250,0.6); }

	.day-label {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
	}
	.day-header.today    .day-label { color: #fbbf24; }
	.day-header.tomorrow .day-label { color: #60a5fa; }

	.day-badge {
		font-size: 10px;
		font-weight: 600;
		padding: 1px 6px;
		border-radius: 999px;
	}
	.day-badge.today    { background: rgba(251,191,36,0.2); color: #fbbf24; }
	.day-badge.tomorrow { background: rgba(96,165,250,0.2); color: #60a5fa; }

	.event {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 5px 8px;
		background: rgba(255,255,255,0.02);
		border-left: 3px solid transparent;
		font-size: 13px;
	}
	.event:last-child { border-radius: 0 0 6px 6px; }
	.event.today    { border-left-color: rgba(251,191,36,0.3); background: rgba(251,191,36,0.04); }
	.event.tomorrow { border-left-color: rgba(96,165,250,0.25); background: rgba(96,165,250,0.03); }

	.event-time {
		font-size: 11px;
		font-weight: 500;
		color: var(--primary);
		white-space: nowrap;
		flex-shrink: 0;
		min-width: 52px;
		padding-top: 2px;
	}
	.event-time.allday {
		color: var(--text-muted);
		font-weight: 400;
	}

	.event-content {
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
		flex: 1;
	}

	.event-title-row {
		display: flex;
		align-items: center;
		gap: 5px;
		min-width: 0;
	}

	.cal-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.event-summary {
		color: var(--text);
		font-size: 13px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.event-cal-name {
		font-size: 10px;
		font-weight: 600;
		opacity: 0.8;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.event-location {
		font-size: 11px;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
