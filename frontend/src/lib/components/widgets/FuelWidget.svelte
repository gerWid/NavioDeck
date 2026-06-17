<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, FuelStation, FuelPrice, SelectedStation } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	type DisplayStation = {
		id: string;
		label: string;
		sublabel: string;
		dist: number | null;
		e5: number;
		e10: number;
		diesel: number;
		is_open: boolean;
	};

	let stations   = $state<DisplayStation[]>([]);
	let location   = $state('');
	let error      = $state('');
	let loading    = $state(true);

	const apiKey       = $derived((widget.config?.api_key as string) || '');
	const cfgLocation  = $derived((widget.config?.location as string) || '');
	const radius       = $derived(Number(widget.config?.radius) || 5);
	const maxStations  = $derived(Number(widget.config?.max_stations) || 5);
	const sortBy       = $derived((widget.config?.sort as string) || 'dist');
	const showE5       = $derived(widget.config?.show_e5 !== false);
	const showE10      = $derived(widget.config?.show_e10 !== false);
	const showDiesel   = $derived(widget.config?.show_diesel !== false);
	const stationSize  = $derived((widget.config?.station_size as string) || 'medium');

	const selectedStations = $derived((): SelectedStation[] => {
		const sel = widget.config?.selected_stations;
		return Array.isArray(sel) && sel.length > 0 ? sel as SelectedStation[] : [];
	});

	const useSelection = $derived(selectedStations().length > 0);

	function cheapest(field: 'e5' | 'e10' | 'diesel'): number {
		const prices = stations.map(s => s[field]).filter(p => p > 0);
		return prices.length ? Math.min(...prices) : 0;
	}

	const cheapE5     = $derived(cheapest('e5'));
	const cheapE10    = $derived(cheapest('e10'));
	const cheapDiesel = $derived(cheapest('diesel'));

	async function fetchData() {
		// The API key may be left blank in the widget config — the server then
		// falls back to the TANKERKOENIG_API_KEY env var. Only the location is
		// required client-side.
		if (!cfgLocation) {
			loading = false;
			error = 'Kein Ort konfiguriert';
			return;
		}
		loading = true;
		error = '';

		try {
			if (useSelection) {
				const ids = selectedStations().map(s => s.id);
				const data = await api.getFuelPrices(apiKey, ids);
				location = cfgLocation;
				stations = selectedStations().map(sel => {
					const p: FuelPrice | undefined = data.prices[sel.id];
					return {
						id:       sel.id,
						label:    sel.caption?.trim() || sel.brand || sel.name,
						sublabel: `${sel.street}, ${sel.city}`,
						dist:     null,
						e5:       p?.e5 ?? 0,
						e10:      p?.e10 ?? 0,
						diesel:   p?.diesel ?? 0,
						is_open:  p?.is_open ?? false,
					};
				});
			} else {
				const data = await api.getFuel({ apiKey, location: cfgLocation, radius, sort: sortBy, max: maxStations });
				location = data.location;
				stations = data.stations.map(s => ({
					id:       s.id,
					label:    s.brand || s.name,
					sublabel: `${s.street}, ${s.city}`,
					dist:     s.dist,
					e5:       s.e5,
					e10:      s.e10,
					diesel:   s.diesel,
					is_open:  s.is_open,
				}));
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Ladefehler';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchData();
		const interval = setInterval(fetchData, 10 * 60 * 1000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		apiKey; cfgLocation; radius; sortBy; maxStations;
		widget.config?.selected_stations;
		fetchData();
	});

	function fmtPrice(p: number): string {
		if (!p || p <= 0) return '–';
		return p.toFixed(3).replace('.', ',') + ' €';
	}

	function isCheapest(price: number, min: number): boolean {
		return price > 0 && min > 0 && price === min;
	}
</script>

{#if loading}
	<div class="center muted">Lade Preise…</div>
{:else if error}
	<div class="center error">{error}</div>
{:else if stations.length === 0}
	<div class="center muted">Keine Tankstellen gefunden</div>
{:else}
	<div class="fuel">
		<div class="location-row">
			<span class="location-label">⛽ {location}</span>
			{#if !useSelection}<span class="radius-label">{radius} km</span>{/if}
		</div>

		{#if (showE5 || showE10 || showDiesel) && stationSize !== 'small'}
			<div class="best-prices">
				{#if showE5 && cheapE5 > 0}
					<div class="best-price">
						<span class="fuel-type">E5</span>
						<span class="price">{fmtPrice(cheapE5)}</span>
					</div>
				{/if}
				{#if showE10 && cheapE10 > 0}
					<div class="best-price">
						<span class="fuel-type">E10</span>
						<span class="price">{fmtPrice(cheapE10)}</span>
					</div>
				{/if}
				{#if showDiesel && cheapDiesel > 0}
					<div class="best-price">
						<span class="fuel-type">Diesel</span>
						<span class="price">{fmtPrice(cheapDiesel)}</span>
					</div>
				{/if}
			</div>
		{/if}

		<div class="station-list">
			{#each stations as s (s.id)}
				<div class="station size-{stationSize}" class:closed={!s.is_open}>
					{#if stationSize === 'small'}
						<span class="s-brand">{s.label}</span>
						<div class="prices-inline">
							{#if showE5 && s.e5 > 0}
								<span class:cheapest={isCheapest(s.e5, cheapE5)}>{fmtPrice(s.e5)}</span>
							{/if}
							{#if showE10 && s.e10 > 0}
								<span class:cheapest={isCheapest(s.e10, cheapE10)}>{fmtPrice(s.e10)}</span>
							{/if}
							{#if showDiesel && s.diesel > 0}
								<span class:cheapest={isCheapest(s.diesel, cheapDiesel)}>{fmtPrice(s.diesel)}</span>
							{/if}
						</div>
					{:else}
						<div class="station-left">
							<span class="brand">{s.label}</span>
							<span class="addr">
								{s.sublabel}{s.dist !== null ? ` · ${s.dist.toFixed(1)} km` : ''}
								{#if !s.is_open}<span class="closed-tag">geschl.</span>{/if}
							</span>
						</div>
						<div class="prices-right">
							{#if showE5 && s.e5 > 0}
								<div class="prow" class:cheapest={isCheapest(s.e5, cheapE5)}>
									<span class="pt">E5</span>
									<span class="pv">{fmtPrice(s.e5)}</span>
								</div>
							{/if}
							{#if showE10 && s.e10 > 0}
								<div class="prow" class:cheapest={isCheapest(s.e10, cheapE10)}>
									<span class="pt">E10</span>
									<span class="pv">{fmtPrice(s.e10)}</span>
								</div>
							{/if}
							{#if showDiesel && s.diesel > 0}
								<div class="prow" class:cheapest={isCheapest(s.diesel, cheapDiesel)}>
									<span class="pt">Di</span>
									<span class="pv">{fmtPrice(s.diesel)}</span>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
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

	.fuel {
		display: flex;
		flex-direction: column;
		gap: 8px;
		height: 100%;
	}

	.location-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 12px;
		color: var(--text-muted);
	}
	.location-label { font-weight: 600; }

	.best-prices {
		display: flex;
		gap: 6px;
	}

	.best-price {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 5px 4px;
		border-radius: 7px;
		background: rgba(255,255,255,0.04);
		border: 1px solid rgba(255,255,255,0.08);
	}

	.fuel-type {
		font-size: 9px;
		font-weight: 700;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.price {
		font-size: 14px;
		font-weight: 700;
		color: #4ade80;
	}

	.station-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
		overflow-y: auto;
		flex: 1;
	}

	/* --- shared station base --- */
	.station {
		border-radius: 8px;
		background: rgba(255,255,255,0.03);
		border: 1px solid rgba(255,255,255,0.06);
	}
	.station.closed { opacity: 0.5; }

	/* --- small: single compact row --- */
	.station.size-small {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 5px 8px;
	}

	.s-brand {
		flex: 1;
		font-size: 11px;
		font-weight: 600;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.prices-inline {
		display: flex;
		gap: 8px;
		flex-shrink: 0;
	}

	.prices-inline span {
		font-size: 11px;
		color: var(--text);
	}

	.prices-inline span.cheapest {
		color: #4ade80;
		font-weight: 700;
	}

	/* --- medium / large: two-column row --- */
	.station.size-medium,
	.station.size-large {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 6px 8px;
	}

	.station.size-large {
		padding: 9px 12px;
	}

	.station-left {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.brand {
		font-size: 12px;
		font-weight: 600;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.station.size-large .brand {
		font-size: 13px;
	}

	.addr {
		font-size: 10px;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.station.size-large .addr {
		font-size: 11px;
	}

	.closed-tag {
		font-size: 9px;
		padding: 1px 4px;
		border-radius: 3px;
		background: rgba(248,113,113,0.15);
		color: #f87171;
		flex-shrink: 0;
	}

	.prices-right {
		display: flex;
		flex-direction: column;
		gap: 2px;
		flex-shrink: 0;
		align-items: flex-end;
	}

	.prow {
		display: flex;
		align-items: baseline;
		gap: 5px;
	}

	.pt {
		font-size: 9px;
		color: var(--text-muted);
		font-weight: 700;
		text-transform: uppercase;
		min-width: 20px;
		text-align: right;
	}

	.pv {
		font-size: 12px;
		color: var(--text);
		min-width: 60px;
		text-align: right;
		font-variant-numeric: tabular-nums;
	}

	.station.size-large .pt {
		font-size: 10px;
	}

	.station.size-large .pv {
		font-size: 13px;
	}

	.prow.cheapest .pv {
		color: #4ade80;
		font-weight: 700;
	}
</style>
