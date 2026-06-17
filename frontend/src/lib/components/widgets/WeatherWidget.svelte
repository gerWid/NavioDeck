<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Widget, WeatherData } from '$lib/types';

	let { widget }: { widget: Widget } = $props();

	let weather = $state<WeatherData | null>(null);
	let error = $state('');
	let loading = $state(true);

	const city         = $derived((widget.config?.city as string) || 'Berlin');
	const units        = $derived((widget.config?.units as string) || 'celsius');
	const unitSymbol   = $derived(units === 'fahrenheit' ? '°F' : '°C');
	const forecastSize = $derived((widget.config?.forecast_size as string) || 'normal');
	const forecastDays = $derived(Number(widget.config?.forecast_days) || 7);

	// optional current-weather fields (off by default unless explicitly enabled)
	const showUV            = $derived(!!widget.config?.show_uv);
	const showPressure      = $derived(!!widget.config?.show_pressure);
	const showVisibility    = $derived(!!widget.config?.show_visibility);
	const showWindDirection = $derived(!!widget.config?.show_wind_direction);
	const showWindGusts     = $derived(!!widget.config?.show_wind_gusts);
	const showDewPoint      = $derived(!!widget.config?.show_dew_point);
	const showCloudCover    = $derived(!!widget.config?.show_cloud_cover);
	const showPrecipitation = $derived(!!widget.config?.show_precipitation);

	// optional forecast extras — precip_prob inherits from large mode when not explicitly set
	const showPrecipProb    = $derived(
		widget.config?.show_precip_prob === undefined
			? forecastSize === 'large'
			: !!widget.config.show_precip_prob
	);
	const showPrecipSum     = $derived(!!widget.config?.show_precip_sum);
	const showUVMax         = $derived(!!widget.config?.show_uv_max);
	const showWindMax       = $derived(!!widget.config?.show_wind_max);
	const showSunriseSunset = $derived(!!widget.config?.show_sunrise_sunset);

	function dayLabel(dateStr: string, index: number): string {
		if (index === 0) return 'Heute';
		const d = new Date(dateStr);
		return d.toLocaleDateString('de-DE', { weekday: 'short' });
	}

	function windCardinal(deg: number): string {
		const dirs = ['N', 'NO', 'O', 'SO', 'S', 'SW', 'W', 'NW'];
		return dirs[Math.round(deg / 45) % 8];
	}

	function uvLabel(uv: number): string {
		if (uv < 3) return 'niedrig';
		if (uv < 6) return 'mittel';
		if (uv < 8) return 'hoch';
		if (uv < 11) return 'sehr hoch';
		return 'extrem';
	}

	async function fetchWeather() {
		loading = true;
		error = '';
		try {
			weather = await api.getWeather(city, units, forecastDays);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Fehler beim Laden';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		fetchWeather();
		const interval = setInterval(fetchWeather, 30 * 60 * 1000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		city; units; forecastDays;
		fetchWeather();
	});
</script>

{#if loading}
	<div class="center muted">Lade Wetter…</div>
{:else if error}
	<div class="center error">{error}</div>
{:else if weather}
	<div class="weather">
		<div class="location">{weather.city}, {weather.country}</div>
		<div class="current">
			<span class="icon">{weather.icon}</span>
			<div class="temp-block">
				<span class="temp">{Math.round(weather.temperature)}{unitSymbol}</span>
				<span class="desc">{weather.description}</span>
			</div>
		</div>

		<div class="meta">
			<span>💧 {weather.humidity}%</span>
			<span>💨 {Math.round(weather.wind_speed)} km/h</span>
			<span>🌡️ {Math.round(weather.feels_like)}{unitSymbol}</span>
			{#if showWindDirection}
				<span>🧭 {windCardinal(weather.wind_direction)} ({weather.wind_direction}°)</span>
			{/if}
			{#if showWindGusts}
				<span>💨↑ {Math.round(weather.wind_gusts)} km/h</span>
			{/if}
			{#if showUV}
				<span>☀ UV {weather.uv_index.toFixed(1)} <span class="uv-label">({uvLabel(weather.uv_index)})</span></span>
			{/if}
			{#if showPressure}
				<span>📊 {Math.round(weather.pressure)} hPa</span>
			{/if}
			{#if showVisibility}
				<span>👁 {weather.visibility.toFixed(1)} km</span>
			{/if}
			{#if showDewPoint}
				<span>🌫️ {Math.round(weather.dew_point)}{unitSymbol}</span>
			{/if}
			{#if showCloudCover}
				<span>☁ {weather.cloud_cover}%</span>
			{/if}
			{#if showPrecipitation}
				<span>🌧 {weather.precipitation} mm</span>
			{/if}
		</div>

		{#if weather.forecast.length > 0}
			<div class="forecast forecast-{forecastSize}">
				{#each weather.forecast as day, i (day.date)}
					<div class="forecast-day">
						<span class="day-label">{dayLabel(day.date, i)}</span>
						<span class="day-icon">{day.icon}</span>
						<span class="day-temps">
							<span class="high">{Math.round(day.temp_max)}°</span>
							<span class="low">{Math.round(day.temp_min)}°</span>
						</span>
						{#if showPrecipProb}
							<span class="day-extra">🌧 {Math.round(day.precip_prob)}%</span>
						{/if}
						{#if showPrecipSum}
							<span class="day-extra">{day.precip_sum.toFixed(1)} mm</span>
						{/if}
						{#if showUVMax}
							<span class="day-extra">UV {day.uv_index_max.toFixed(1)}</span>
						{/if}
						{#if showWindMax}
							<span class="day-extra">💨 {Math.round(day.wind_max)}</span>
						{/if}
						{#if showSunriseSunset}
							<span class="day-extra sunrise">🌅{day.sunrise}</span>
							<span class="day-extra sunset">🌇{day.sunset}</span>
						{/if}
					</div>
				{/each}
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
	}

	.muted { color: var(--text-muted); }
	.error { color: #f87171; }

	.weather {
		display: flex;
		flex-direction: column;
		gap: 10px;
		height: 100%;
	}

	.location {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
		letter-spacing: 0.04em;
	}

	.current {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.icon {
		font-size: 42px;
		line-height: 1;
	}

	.temp-block {
		display: flex;
		flex-direction: column;
	}

	.temp {
		font-size: 32px;
		font-weight: 700;
		line-height: 1;
		color: var(--text);
	}

	.desc {
		font-size: 12px;
		color: var(--text-muted);
		margin-top: 2px;
	}

	.meta {
		display: flex;
		gap: 10px;
		font-size: 12px;
		color: var(--text-muted);
		flex-wrap: wrap;
	}

	.uv-label {
		font-size: 10px;
	}

	.forecast {
		display: flex;
		justify-content: space-between;
		gap: 2px;
		margin-top: auto;
		padding-top: 8px;
		border-top: 1px solid rgba(255,255,255,0.07);
		overflow-x: auto;
	}

	.forecast-day {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		flex: 1;
		min-width: 0;
	}

	/* compact */
	.forecast-compact .day-label { font-size: 10px; color: var(--text-muted); font-weight: 500; }
	.forecast-compact .day-icon  { font-size: 16px; }
	.forecast-compact .day-temps { font-size: 10px; }
	.forecast-compact .day-extra { font-size: 9px; color: var(--text-muted); }

	/* normal (default) */
	.forecast-normal .day-label { font-size: 11px; color: var(--text-muted); font-weight: 500; }
	.forecast-normal .day-icon  { font-size: 22px; }
	.forecast-normal .day-temps { font-size: 11px; }
	.forecast-normal .day-extra { font-size: 10px; color: var(--text-muted); }

	/* large */
	.forecast-large .day-label { font-size: 12px; color: var(--text-muted); font-weight: 600; }
	.forecast-large .day-icon  { font-size: 30px; }
	.forecast-large .day-temps { font-size: 12px; }
	.forecast-large .day-extra { font-size: 11px; color: var(--text-muted); }

	.day-label { font-size: 11px; color: var(--text-muted); font-weight: 500; }

	.day-temps {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.high { color: var(--text); font-weight: 600; }
	.low  { color: var(--text-muted); }

	.day-extra {
		font-size: 10px;
		color: var(--text-muted);
		white-space: nowrap;
	}

	.sunrise { color: #fbbf24; }
	.sunset  { color: #f97316; }
</style>
