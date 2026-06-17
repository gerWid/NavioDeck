<script lang="ts">
	import { dashboard } from '$lib/stores/dashboard.svelte';
	import type { Widget, WidgetItem, WidgetStyle, FuelStation, SelectedStation, CalendarSource, RssSource } from '$lib/types';
	import { api } from '$lib/api';
	import CitySearch from './CitySearch.svelte';
	import WallpaperPicker from './WallpaperPicker.svelte';
	import IconPicker from './IconPicker.svelte';

	let widget = $state<Widget>((() => {
		const w = JSON.parse(JSON.stringify(dashboard.editingWidget!));
		w.style  = w.style  ?? {};
		w.config = w.config ?? {};
		// Migrate old single-source calendar config to sources array
		if (w.type === 'calendar' && w.config.source && !Array.isArray(w.config.sources)) {
			w.config.sources = [{ name: 'Kalender', url: w.config.source, color: '#818cf8' }];
			delete w.config.source;
		}
		return w;
	})());

	function readPanelWidth() {
		try { return parseInt(localStorage.getItem('editor-panel-width') || '420', 10) || 420; }
		catch { return 420; }
	}

	let panelWidth    = $state(readPanelWidth());
	let resizeDragging = $state(false);

	function startResize(e: PointerEvent) {
		resizeDragging = true;
		const startX = e.clientX;
		const startW = panelWidth;
		const onMove = (ev: PointerEvent) => {
			panelWidth = Math.max(300, Math.min(window.innerWidth * 0.9, startW + (startX - ev.clientX)));
		};
		const onUp = () => {
			resizeDragging = false;
			try { localStorage.setItem('editor-panel-width', String(panelWidth)); } catch { /* */ }
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
		e.preventDefault();
	}

	// Tracks whether the most recent pointerdown started inside the panel.
	// Prevents accidental save() when the user drags from inside the panel
	// to the overlay (browser fires click on overlay in that case).
	let _pointerDownInPanel = false;

	// index-based: true = expanded, false = collapsed; false = existing, true = newly added draft
	let expanded  = $state<boolean[]>((widget.items ?? []).map(() => false));
	let isDraft   = $state<boolean[]>((widget.items ?? []).map(() => false));
	// Stable key per item so {#each} tracks items through reorders instead of by index.
	// Without this Svelte reuses the wrong DOM nodes after a move and inputs show stale values.
	let itemKeys  = $state<string[]>((widget.items ?? []).map((_, i) => `k${i}-${Math.random()}`));
	// Explicit position values bound to the inputs; reset after every move so every
	// input always shows the correct current position number.
	let posInputs = $state<number[]>((widget.items ?? []).map((_, i) => i + 1));

	function toggle(i: number) {
		expanded[i] = !expanded[i];
		expanded = [...expanded];
	}

	function discard() {
		(document.activeElement as HTMLElement | null)?.blur();
		dashboard.editingWidget = null;
	}

	async function save() {
		(document.activeElement as HTMLElement | null)?.blur();
		if (widget.style) {
			const cleaned: Record<string, string> = {};
			for (const [k, v] of Object.entries(widget.style)) {
				if (v) cleaned[k] = v;
			}
			widget.style = Object.keys(cleaned).length > 0 ? (cleaned as WidgetStyle) : undefined;
		}
		// Move draft (newly added) items to the end
		if (widget.items && isDraft.some(Boolean)) {
			const pairs = widget.items.map((item, i) => ({ item, draft: isDraft[i] ?? false }));
			widget.items = [
				...pairs.filter(p => !p.draft).map(p => p.item),
				...pairs.filter(p =>  p.draft).map(p => p.item),
			];
		}
		await dashboard.updateWidget(widget);
	}

	function setStyle(key: keyof WidgetStyle, val: string) {
		widget.style = { ...(widget.style ?? {}), [key]: val };
	}

	function clearStyle(key: keyof WidgetStyle) {
		if (!widget.style) return;
		const s = { ...widget.style } as Record<string, string>;
		delete s[key];
		widget.style = s as WidgetStyle;
	}

	function addItem() {
		widget.items = [{ name: '', url: '', icon: '', description: '' }, ...(widget.items ?? [])];
		expanded  = [true, ...expanded];
		isDraft   = [true, ...isDraft];
		itemKeys  = [`k-new-${Date.now()}`, ...itemKeys];
		posInputs = (widget.items ?? []).map((_, i) => i + 1);
	}

	function removeItem(index: number) {
		widget.items = widget.items?.filter((_, i) => i !== index);
		expanded  = expanded.filter((_, i) => i !== index);
		isDraft   = isDraft.filter((_, i) => i !== index);
		itemKeys  = itemKeys.filter((_, i) => i !== index);
		posInputs = (widget.items ?? []).map((_, i) => i + 1);
	}

	function moveItem(index: number, direction: -1 | 1) {
		if (!widget.items) return;
		const items = [...widget.items];
		const target = index + direction;
		if (target < 0 || target >= items.length) return;
		[items[index], items[target]] = [items[target], items[index]];
		widget.items = items;
		[expanded[index], expanded[target]] = [expanded[target], expanded[index]];
		expanded = [...expanded];
		[isDraft[index], isDraft[target]] = [isDraft[target], isDraft[index]];
		isDraft = [...isDraft];
		[itemKeys[index], itemKeys[target]] = [itemKeys[target], itemKeys[index]];
		itemKeys  = [...itemKeys];
		posInputs = items.map((_, i) => i + 1);
	}

	function moveItemTo(fromIndex: number, toIndex: number) {
		if (!widget.items) return;
		const clamped = Math.max(0, Math.min(widget.items.length - 1, toIndex));
		// Always reset so the input shows its actual position even when no move happens
		if (clamped === fromIndex) {
			posInputs = widget.items.map((_, i) => i + 1);
			return;
		}
		const items = [...widget.items];
		const exp   = [...expanded];
		const draft = [...isDraft];
		const keys  = [...itemKeys];
		const [item]     = items.splice(fromIndex, 1);
		const [expVal]   = exp.splice(fromIndex, 1);
		const [draftVal] = draft.splice(fromIndex, 1);
		const [keyVal]   = keys.splice(fromIndex, 1);
		items.splice(clamped, 0, item);
		exp.splice(clamped, 0, expVal);
		draft.splice(clamped, 0, draftVal);
		keys.splice(clamped, 0, keyVal);
		widget.items = items;
		expanded  = exp;
		isDraft   = draft;
		itemKeys  = keys;
		posInputs = items.map((_, i) => i + 1);
	}

	function applyToAll(index: number) {
		if (!widget.items) return;
		const src = widget.items[index];
		widget.items = widget.items.map((it) => ({
			...it,
			size:           src.size,
			item_width:     src.item_width,
			item_height:    src.item_height,
			icon_size:      src.icon_size,
			name_font_size: src.name_font_size,
			desc_font_size: src.desc_font_size,
			font_family:    src.font_family,
			text_align:     src.text_align,
			text_color:     src.text_color,
			desc_color:     src.desc_color,
			bg_color:       src.bg_color,
			border_color:   src.border_color,
			icon_bg_color:  src.icon_bg_color,
		}));
	}

	function clearAllColors() {
		const img = widget.style?.background_image;
		widget.style = img ? { background_image: img } : undefined;
	}

	function clearItemColors(i: number) {
		if (!widget.items) return;
		widget.items[i].text_color    = undefined;
		widget.items[i].desc_color    = undefined;
		widget.items[i].bg_color      = undefined;
		widget.items[i].border_color  = undefined;
		widget.items[i].icon_bg_color = undefined;
		widget.items = [...widget.items];
	}

	const ALIGN_OPTIONS = [
		{ value: 'left',   label: '←', title: 'Linksbündig' },
		{ value: 'center', label: '↔', title: 'Zentriert' },
		{ value: 'right',  label: '→', title: 'Rechtsbündig' },
	];

	// Returns "#rrggbb" fallback for a color input when field is unset
	function cpVal(v: string | undefined, fallback: string): string {
		return v && v.startsWith('#') ? v : fallback;
	}

	function colorInput(
		getter: () => string | undefined,
		setter: (v: string | undefined) => void
	) {
		return {
			pickerVal: cpVal(getter(), '#f1f5f9'),
			textVal:   getter() ?? '',
			onPickerInput: (e: Event) => setter((e.target as HTMLInputElement).value),
			onTextInput:   (e: Event) => setter((e.target as HTMLInputElement).value || undefined),
		};
	}

	const FONTS = ['', 'Inter', 'Roboto', 'DM Sans', 'Nunito', 'Fira Code', 'JetBrains Mono'];

	function numInput(val: number | undefined, set: (v: number | undefined) => void) {
		return (e: Event) => {
			const v = parseInt((e.target as HTMLInputElement).value);
			set(isNaN(v) ? undefined : v);
		};
	}

	// --- Fuel station picker ---
	let fuelSearchResults = $state<FuelStation[]>([]);
	let fuelSearchLoading = $state(false);
	let fuelSearchError   = $state('');

	async function searchFuelStations() {
		const apiKey   = (widget.config?.api_key as string) || '';
		const location = widget.config?.location as string;
		// The key may be left blank — the server falls back to its config.yaml key.
		if (!location) {
			fuelSearchError = 'Bitte zuerst einen Ort eingeben.';
			return;
		}
		fuelSearchLoading = true;
		fuelSearchError = '';
		fuelSearchResults = [];
		try {
			const data = await api.getFuel({ apiKey, location, radius: Number(widget.config?.radius) || 5, max: 25 });
			fuelSearchResults = data.stations;
		} catch (e) {
			fuelSearchError = e instanceof Error ? e.message : 'Fehler';
		} finally {
			fuelSearchLoading = false;
		}
	}

	function getSelectedStations(): SelectedStation[] {
		const sel = widget.config?.selected_stations;
		return Array.isArray(sel) ? sel as SelectedStation[] : [];
	}

	function isStationSelected(id: string): boolean {
		return getSelectedStations().some(s => s.id === id);
	}

	function toggleStation(station: FuelStation) {
		if (isStationSelected(station.id)) {
			widget.config!.selected_stations = getSelectedStations().filter(s => s.id !== station.id);
		} else {
			widget.config!.selected_stations = [
				...getSelectedStations(),
				{ id: station.id, caption: '', name: station.name, brand: station.brand, street: station.street, city: station.city } satisfies SelectedStation,
			];
		}
	}

	function updateCaption(id: string, caption: string) {
		widget.config!.selected_stations = getSelectedStations().map(s => s.id === id ? { ...s, caption } : s);
	}

	function removeStation(id: string) {
		widget.config!.selected_stations = getSelectedStations().filter(s => s.id !== id);
	}

	function moveStation(index: number, dir: -1 | 1) {
		const sel = [...getSelectedStations()];
		const target = index + dir;
		if (target < 0 || target >= sel.length) return;
		[sel[index], sel[target]] = [sel[target], sel[index]];
		widget.config!.selected_stations = sel;
	}
	// --- end fuel station picker ---

	const typeLabel: Record<string, string> = {
		services:  'Services',
		weather:   'Wetter',
		clock:     'Uhr',
		bookmarks: 'Lesezeichen',
		news:      'Nachrichten',
	};

	const BUNDESLAENDER = [
		{ id: 1,  name: 'Baden-Württemberg' },
		{ id: 2,  name: 'Bayern' },
		{ id: 3,  name: 'Berlin' },
		{ id: 4,  name: 'Brandenburg' },
		{ id: 5,  name: 'Bremen' },
		{ id: 6,  name: 'Hamburg' },
		{ id: 7,  name: 'Hessen' },
		{ id: 8,  name: 'Mecklenburg-Vorpommern' },
		{ id: 9,  name: 'Niedersachsen' },
		{ id: 10, name: 'Nordrhein-Westfalen' },
		{ id: 11, name: 'Rheinland-Pfalz' },
		{ id: 12, name: 'Saarland' },
		{ id: 13, name: 'Sachsen' },
		{ id: 14, name: 'Sachsen-Anhalt' },
		{ id: 15, name: 'Schleswig-Holstein' },
		{ id: 16, name: 'Thüringen' },
	];

	function newsRegions(): number[] {
		const r = widget.config?.regions;
		return Array.isArray(r) ? r : [];
	}

	function toggleRegion(id: number) {
		const current = newsRegions();
		widget.config!.regions = current.includes(id)
			? current.filter((r: number) => r !== id)
			: [...current, id].sort((a, b) => a - b);
	}

	// --- RSS source helpers ---
	const RSS_PRESETS: RssSource[] = [
		{ name: 'Tagesschau',    url: 'https://www.tagesschau.de/xml/rss2' },
		{ name: 'Spiegel',       url: 'https://www.spiegel.de/schlagzeilen/index.rss' },
		{ name: 'Zeit Online',   url: 'https://newsfeed.zeit.de/all' },
		{ name: 'Heise',         url: 'https://www.heise.de/rss/heise.rdf' },
		{ name: 'Golem',         url: 'https://rss.golem.de/rss.php?feed=ATOM1.0' },
		{ name: 'FAZ',           url: 'https://www.faz.net/rss/aktuell/' },
		{ name: 'Focus',         url: 'https://rss.focus.de/fol/xml/rss_folnews.xml' },
		{ name: 'SZ',            url: 'https://rss.sueddeutsche.de/rss/Topthemen' },
		{ name: 'Welt',          url: 'https://www.welt.de/feeds/latest.rss' },
		{ name: 'n-tv',          url: 'https://www.n-tv.de/rss' },
	];

	function getRssSources(): RssSource[] {
		return Array.isArray(widget.config?.rss_sources) ? (widget.config!.rss_sources as RssSource[]) : [];
	}

	function addRssSource(preset?: RssSource) {
		const already = getRssSources();
		widget.config!.rss_sources = [...already, preset ? { ...preset } : { name: '', url: '' }];
	}

	function removeRssSource(i: number) {
		widget.config!.rss_sources = getRssSources().filter((_, idx) => idx !== i);
	}

	function updateRssSource(i: number, field: keyof RssSource, value: string) {
		widget.config!.rss_sources = getRssSources().map((s, idx) => idx === i ? { ...s, [field]: value } : s);
	}

	function isRssPresetAdded(preset: RssSource): boolean {
		return getRssSources().some(s => s.url === preset.url);
	}

	// --- Calendar multi-source helpers ---
	const CAL_COLORS = ['#818cf8','#fb923c','#34d399','#f472b6','#60a5fa','#a78bfa','#fbbf24'];

	function getCalSources(): CalendarSource[] {
		return Array.isArray(widget.config?.sources) ? (widget.config!.sources as CalendarSource[]) : [];
	}

	function addCalSource() {
		const srcs = getCalSources();
		widget.config!.sources = [
			...srcs,
			{ name: '', url: '', color: CAL_COLORS[srcs.length % CAL_COLORS.length] },
		];
	}

	function removeCalSource(i: number) {
		widget.config!.sources = getCalSources().filter((_, idx) => idx !== i);
	}

	function updateCalSource(i: number, field: keyof CalendarSource, value: string) {
		const srcs = getCalSources().map((s, idx) => idx === i ? { ...s, [field]: value } : s);
		widget.config!.sources = srcs;
	}
</script>

<div class="panel-overlay"
	onpointerdown={() => { _pointerDownInPanel = false; }}
	onclick={() => { if (!_pointerDownInPanel) save(); _pointerDownInPanel = false; }}
	onkeydown={(e) => { if (e.key === 'Escape') save(); }}
	role="dialog" aria-modal="true" tabindex="-1">
	<div class="panel" style="width:{panelWidth}px"
		onpointerdown={(e) => { _pointerDownInPanel = true; e.stopPropagation(); }}
		onclick={(e) => e.stopPropagation()}
		role="presentation">
		<div class="panel-resize-handle" class:dragging={resizeDragging} onpointerdown={startResize} role="separator" tabindex="-1" aria-label="Größe anpassen"></div>
		<div class="panel-header">
			<span>{typeLabel[widget.type] ?? widget.type} bearbeiten</span>
			<button class="close-btn" onclick={save}>✕</button>
		</div>

		<div class="panel-body">
			<div class="form-group">
				<label for="widget-title">Titel</label>
				<input id="widget-title" type="text" bind:value={widget.title} placeholder="Widget-Titel (optional)" />
			</div>

			<!-- Weather config -->
			{#if widget.type === 'weather'}
				<div class="form-group">
					<span class="label-text">Stadt</span>
					<CitySearch bind:city={widget.config!.city} />
				</div>
				<div class="form-group">
					<label for="weather-units">Einheiten</label>
					<select id="weather-units" bind:value={widget.config!.units}>
						<option value="celsius">Celsius (°C)</option>
						<option value="fahrenheit">Fahrenheit (°F)</option>
					</select>
				</div>
				<div class="form-group">
					<label for="weather-forecast-days">Vorhersage-Tage (1–16)</label>
					<input id="weather-forecast-days" type="number" min="1" max="16" placeholder="7"
						value={widget.config!.forecast_days ?? 7}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.forecast_days = isNaN(v) ? 7 : Math.min(16, Math.max(1, v));
						}}
					/>
				</div>
				<div class="form-group">
					<label for="weather-forecast-size">Vorhersage-Größe</label>
					<select id="weather-forecast-size" bind:value={widget.config!.forecast_size}>
						<option value="compact">Kompakt</option>
						<option value="normal">Normal</option>
						<option value="large">Groß</option>
					</select>
				</div>

				<div class="section-mini-title">Aktuelle Wetterdaten</div>
				<div class="weather-toggles">
					<label class="toggle-row">
						<span>Windrichtung</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_wind_direction} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Windböen</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_wind_gusts} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>UV-Index</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_uv} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Luftdruck (hPa)</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_pressure} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Sichtweite (km)</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_visibility} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Taupunkt</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_dew_point} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Bewölkung (%)</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_cloud_cover} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Niederschlag (mm)</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_precipitation} />
							<span class="toggle-track"></span>
						</label>
					</label>
				</div>

				<div class="section-mini-title">Vorhersage-Details</div>
				<div class="weather-toggles">
					<label class="toggle-row">
						<span>Regenwahrscheinlichkeit</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_precip_prob} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Niederschlagsmenge (mm)</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_precip_sum} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>UV-Index max.</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_uv_max} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Max. Windgeschw.</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_wind_max} />
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Sonnenauf-/-untergang</span>
						<label class="toggle">
							<input type="checkbox" bind:checked={widget.config!.show_sunrise_sunset} />
							<span class="toggle-track"></span>
						</label>
					</label>
				</div>
			{/if}

			<!-- Clock config -->
			{#if widget.type === 'clock'}
				<div class="form-group">
					<label for="clock-style">Darstellung</label>
					<select id="clock-style"
						value={widget.config!.style ?? 'digital'}
						onchange={(e) => { widget.config!.style = (e.target as HTMLSelectElement).value; }}
					>
						<option value="digital">Digital</option>
						<option value="analog">Analog</option>
					</select>
				</div>
				<div class="form-group">
					<label for="clock-timezone">Zeitzone</label>
					<input id="clock-timezone" type="text" bind:value={widget.config!.timezone} placeholder="Europe/Berlin" />
				</div>
				{#if (widget.config!.style ?? 'digital') === 'digital'}
					<div class="form-group">
						<label for="clock-format">Format</label>
						<select id="clock-format" bind:value={widget.config!.format}>
							<option value="24h">24 Stunden</option>
							<option value="12h">12 Stunden (AM/PM)</option>
						</select>
					</div>
				{:else}
					<div class="form-group">
						<span class="label-text">Sekundenzeiger</span>
						<label class="toggle">
							<input type="checkbox"
								checked={widget.config!.show_seconds !== false}
								onchange={(e) => { widget.config!.show_seconds = (e.target as HTMLInputElement).checked; }}
							/>
							<span class="toggle-track"></span>
						</label>
					</div>
				{/if}
				<div class="form-group">
					<span class="label-text">Datum anzeigen</span>
					<label class="toggle">
						<input type="checkbox" bind:checked={widget.config!.show_date} />
						<span class="toggle-track"></span>
					</label>
				</div>
			{/if}

			<!-- News config -->
			{#if widget.type === 'news'}
				<div class="form-group">
					<label for="news-mode">Modus</label>
					<select id="news-mode"
						value={widget.config!.mode ?? 'rss'}
						onchange={(e) => { widget.config!.mode = (e.target as HTMLSelectElement).value; }}
					>
						<option value="rss">RSS-Feeds (eigene Quellen)</option>
						<option value="tagesschau">Tagesschau API</option>
					</select>
				</div>

				{#if (widget.config!.mode ?? 'rss') === 'rss'}
					<div class="section-mini-title">RSS-Quellen</div>
					<div class="rss-presets">
						{#each RSS_PRESETS as preset}
							<button
								class="rss-preset-btn"
								class:added={isRssPresetAdded(preset)}
								onclick={() => { if (!isRssPresetAdded(preset)) addRssSource(preset); }}
								title={isRssPresetAdded(preset) ? 'Bereits hinzugefügt' : `${preset.name} hinzufügen`}
								disabled={isRssPresetAdded(preset)}
							>
								{isRssPresetAdded(preset) ? '✓' : '+'} {preset.name}
							</button>
						{/each}
					</div>
					{#each getRssSources() as src, i (i)}
						<div class="rss-source-row">
							<div class="cal-source-fields">
								<input
									type="text"
									placeholder="Name (z.B. Spiegel)"
									value={src.name}
									oninput={(e) => updateRssSource(i, 'name', (e.target as HTMLInputElement).value)}
								/>
								<input
									type="text"
									placeholder="https://www.spiegel.de/schlagzeilen/index.rss"
									value={src.url}
									oninput={(e) => updateRssSource(i, 'url', (e.target as HTMLInputElement).value)}
								/>
							</div>
							<button class="icon-btn-sm danger cal-remove-btn" onclick={() => removeRssSource(i)} title="Entfernen">✕</button>
						</div>
					{/each}
					<button class="btn btn-ghost btn-sm cal-add-btn" onclick={() => addRssSource()}>
						+ Eigene Quelle hinzufügen
					</button>
					<div class="form-group" style="margin-top: 10px">
						<label for="news-max-items">Max. Artikel</label>
						<input id="news-max-items" type="number" min="1" max="100" placeholder="20"
							value={widget.config!.max_items ?? 20}
							oninput={(e) => {
								const v = parseInt((e.target as HTMLInputElement).value);
								if (!isNaN(v) && v > 0) widget.config!.max_items = v;
							}}
						/>
					</div>
				{:else}
					<!-- Tagesschau filters -->
					<div class="form-group">
						<label for="news-ressort">Themengebiet (Ressort)</label>
						<select id="news-ressort" bind:value={widget.config!.ressort}>
							<option value="">Alle Themen</option>
							<option value="inland">Deutschland</option>
							<option value="ausland">Ausland</option>
							<option value="wirtschaft">Wirtschaft</option>
							<option value="sport">Sport</option>
							<option value="video">Video</option>
							<option value="investigativ">Investigativ</option>
							<option value="faktenfinder">Faktenfinder</option>
						</select>
					</div>
					<div class="form-group">
						<span class="label-text">Bundesland (optional)</span>
						<details class="regions-dropdown">
							<summary class="regions-trigger">
								{newsRegions().length === 0
									? 'Alle Bundesländer'
									: newsRegions().length === 1
										? BUNDESLAENDER.find(b => b.id === newsRegions()[0])?.name ?? '1 ausgewählt'
										: `${newsRegions().length} ausgewählt`}
							</summary>
							<div class="regions-list">
								<label class="region-option">
									<input
										type="checkbox"
										checked={newsRegions().length === 0}
										onchange={() => { widget.config!.regions = []; }}
									/>
									Alle Bundesländer
								</label>
								<div class="regions-divider"></div>
								{#each BUNDESLAENDER as bl}
									<label class="region-option">
										<input
											type="checkbox"
											checked={newsRegions().includes(bl.id)}
											onchange={() => toggleRegion(bl.id)}
										/>
										{bl.name}
									</label>
								{/each}
							</div>
						</details>
					</div>
					<div class="form-group">
						<label for="news-page-size">Anzahl Artikel</label>
						<input id="news-page-size" type="number" min="1" max="100" placeholder="10"
							value={widget.config!.page_size ?? 10}
							oninput={(e) => {
								const v = parseInt((e.target as HTMLInputElement).value);
								if (!isNaN(v) && v > 0) widget.config!.page_size = v;
							}}
						/>
					</div>
					<div class="section-mini-title">Rubrik-Badge</div>
					<div class="form-group">
						<label for="news-badge-color">Schrift- / Rahmenfarbe</label>
						<div class="color-row">
							<input id="news-badge-color" type="color" class="color-picker"
								value={widget.style?.topline_color || '#818cf8'}
								oninput={(e) => setStyle('topline_color', (e.target as HTMLInputElement).value)}
							/>
							<input type="text"
								value={widget.style?.topline_color || ''}
								placeholder="Standard (Akzentfarbe)"
								aria-label="Schrift- / Rahmenfarbe als Text"
								oninput={(e) => setStyle('topline_color', (e.target as HTMLInputElement).value)}
							/>
							{#if widget.style?.topline_color}
								<button class="reset-btn" onclick={() => clearStyle('topline_color')} title="Zurücksetzen">✕</button>
							{/if}
						</div>
					</div>
					<div class="form-group">
						<label for="news-badge-bg">Hintergrundfarbe</label>
						<div class="color-row">
							<input id="news-badge-bg" type="color" class="color-picker"
								value={widget.style?.topline_bg || '#1e293b'}
								oninput={(e) => setStyle('topline_bg', (e.target as HTMLInputElement).value)}
							/>
							<input type="text"
								value={widget.style?.topline_bg || ''}
								placeholder="Standard (transparent)"
								aria-label="Hintergrundfarbe als Text"
								oninput={(e) => setStyle('topline_bg', (e.target as HTMLInputElement).value)}
							/>
							{#if widget.style?.topline_bg}
								<button class="reset-btn" onclick={() => clearStyle('topline_bg')} title="Zurücksetzen">✕</button>
							{/if}
						</div>
					</div>
				{/if}

				<div class="form-group">
					<label for="news-refresh-interval">Aktualisierungsintervall</label>
					<select id="news-refresh-interval" bind:value={widget.config!.refresh_interval}>
						<option value={5}>5 Minuten</option>
						<option value={10}>10 Minuten</option>
						<option value={15}>15 Minuten</option>
						<option value={30}>30 Minuten</option>
						<option value={60}>60 Minuten</option>
					</select>
				</div>
			{/if}
			<!-- Docker config -->
			{#if widget.type === 'docker'}
				<div class="form-group">
					<label for="docker-endpoint">Docker-Endpunkt</label>
					<input id="docker-endpoint" type="text" bind:value={widget.config!.endpoint}
						placeholder="unix:///var/run/docker.sock" />
					<span class="hint">Unix-Socket: unix:///var/run/docker.sock · Socket-Proxy: tcp://host:2375 oder http://host.docker.internal:2375</span>
				</div>
				<div class="form-group">
					<span class="label-text">Gestoppte Container anzeigen</span>
					<label class="toggle">
						<input type="checkbox" bind:checked={widget.config!.show_stopped} />
						<span class="toggle-track"></span>
					</label>
				</div>
				<div class="form-group">
					<label for="docker-max">Max. Einträge in der Liste</label>
					<input id="docker-max" type="number" min="1" max="50" placeholder="10"
						value={widget.config!.max_items ?? 10}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.max_items = isNaN(v) ? 10 : v;
						}}
					/>
				</div>
			{/if}

			<!-- Calendar / iCal config -->
			{#if widget.type === 'calendar'}
				<div class="section-mini-title">Kalenderquellen</div>
				{#each getCalSources() as src, i (i)}
					<div class="cal-source-row">
						<div class="cal-color-wrap">
							<input
								type="color"
								class="cal-color-picker"
								value={src.color || CAL_COLORS[i % CAL_COLORS.length]}
								oninput={(e) => updateCalSource(i, 'color', (e.target as HTMLInputElement).value)}
								title="Kalenderfarbe"
							/>
						</div>
						<div class="cal-source-fields">
							<input
								type="text"
								placeholder="Name (z.B. Mein Kalender)"
								value={src.name}
								oninput={(e) => updateCalSource(i, 'name', (e.target as HTMLInputElement).value)}
							/>
							<input
								type="text"
								placeholder="https://calendar.google.com/…/basic.ics"
								value={src.url}
								oninput={(e) => updateCalSource(i, 'url', (e.target as HTMLInputElement).value)}
							/>
						</div>
						<button class="icon-btn-sm danger cal-remove-btn" onclick={() => removeCalSource(i)} title="Entfernen">✕</button>
					</div>
				{/each}
				<button class="btn btn-ghost btn-sm cal-add-btn" onclick={addCalSource}>
					+ Kalender hinzufügen
				</button>
				<span class="hint" style="margin-top:4px;display:block">
					Google Kalender → Einstellungen → Kalender → „Geheime Adresse im iCal-Format"
				</span>
				<div class="form-group" style="margin-top:14px">
					<label for="cal-days">Tage voraus</label>
					<input id="cal-days" type="number" min="1" max="365" placeholder="30"
						value={widget.config!.days_ahead ?? 30}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.days_ahead = isNaN(v) ? 30 : v;
						}}
					/>
				</div>
				<div class="form-group">
					<label for="cal-max">Max. Einträge</label>
					<input id="cal-max" type="number" min="1" max="200" placeholder="20"
						value={widget.config!.max_items ?? 20}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.max_items = isNaN(v) ? 20 : v;
						}}
					/>
				</div>
			{/if}

			<!-- Garbage / iCal config -->
			{#if widget.type === 'garbage'}
				<div class="form-group">
					<label for="garbage-source">iCal-Quelle</label>
					<input id="garbage-source" type="text" bind:value={widget.config!.source}
						placeholder="https://... oder dateiname.ics" />
					<span class="hint">URL (https://) oder lokaler Dateiname im /data-Ordner</span>
				</div>
				<div class="form-group">
					<label for="garbage-days">Tage voraus</label>
					<input id="garbage-days" type="number" min="1" max="365" placeholder="30"
						value={widget.config!.days_ahead ?? 30}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.days_ahead = isNaN(v) ? 30 : v;
						}}
					/>
				</div>
				<div class="form-group">
					<label for="garbage-max">Max. Einträge</label>
					<input id="garbage-max" type="number" min="1" max="50" placeholder="10"
						value={widget.config!.max_items ?? 10}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.max_items = isNaN(v) ? 10 : v;
						}}
					/>
				</div>
				<div class="form-group">
					<span class="label-text">Nur nächsten Termin anzeigen</span>
					<label class="toggle">
						<input type="checkbox" bind:checked={widget.config!.show_next_only} />
						<span class="toggle-track"></span>
					</label>
				</div>
			{/if}

			<!-- Fuel / Tankerkönig config -->
			{#if widget.type === 'fuel'}
				<div class="form-group">
					<label for="fuel-apikey">Tankerkönig API-Key</label>
					<input id="fuel-apikey" type="text" bind:value={widget.config!.api_key}
						placeholder="leer lassen = Server-Key (config.yaml)" />
					<span class="hint">Leer lassen nutzt den serverseitigen Key aus data/config.yaml. Eigener Key: kostenlos unter creativecommons.tankerkoenig.de</span>
				</div>
				<div class="form-group">
					<label for="fuel-location">Ort / Adresse</label>
					<input id="fuel-location" type="text" bind:value={widget.config!.location}
						placeholder="Berlin" />
				</div>
				<div class="form-group">
					<label for="fuel-radius">Suchradius (km)</label>
					<input id="fuel-radius" type="number" min="1" max="25" placeholder="5"
						value={widget.config!.radius ?? 5}
						oninput={(e) => {
							const v = parseFloat((e.target as HTMLInputElement).value);
							widget.config!.radius = isNaN(v) ? 5 : v;
						}}
					/>
				</div>

				<!-- Station picker -->
				<div class="section-mini-title">Tankstellen auswählen</div>
				<span class="hint" style="margin-bottom:6px;display:block">
					Ohne Auswahl werden automatisch die nächsten Stationen angezeigt.
				</span>
				<button class="btn btn-ghost btn-sm fuel-search-btn"
					onclick={searchFuelStations}
					disabled={fuelSearchLoading}
				>
					{fuelSearchLoading ? '⏳ Suche…' : '🔍 Stationen suchen'}
				</button>
				{#if fuelSearchError}
					<span class="hint" style="color:#f87171;margin-top:4px;display:block">{fuelSearchError}</span>
				{/if}

				{#if fuelSearchResults.length > 0}
					<div class="fuel-results">
						{#each fuelSearchResults as st (st.id)}
							<label class="fuel-result-row" class:selected={isStationSelected(st.id)}>
								<input type="checkbox"
									checked={isStationSelected(st.id)}
									onchange={() => toggleStation(st)}
								/>
								<div class="fuel-result-info">
									<span class="fuel-result-name">{st.brand || st.name}</span>
									<span class="fuel-result-addr">{st.street}, {st.city}</span>
								</div>
								<span class="fuel-result-dist">{st.dist.toFixed(1)} km</span>
							</label>
						{/each}
					</div>
				{/if}

				{#if getSelectedStations().length > 0}
					<div class="section-mini-title" style="margin-top:10px">Ausgewählte Tankstellen</div>
					<div class="fuel-selected-list">
						{#each getSelectedStations() as sel, i (sel.id)}
							<div class="fuel-sel-row">
								<div class="fuel-sel-meta">
									<span class="fuel-sel-name">{sel.brand || sel.name}</span>
									<span class="fuel-sel-addr">{sel.street}, {sel.city}</span>
								</div>
								<input
									type="text"
									class="fuel-sel-caption"
									placeholder="Bezeichnung (optional)"
									value={sel.caption}
									oninput={(e) => updateCaption(sel.id, (e.target as HTMLInputElement).value)}
								/>
								<div class="fuel-sel-actions">
									<button class="icon-btn-sm" onclick={() => moveStation(i, -1)} title="Nach oben">↑</button>
									<button class="icon-btn-sm" onclick={() => moveStation(i, 1)} title="Nach unten">↓</button>
									<button class="icon-btn-sm danger" onclick={() => removeStation(sel.id)} title="Entfernen">✕</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}

				<div class="section-mini-title" style="margin-top:10px">Sortierung &amp; Anzeige</div>
				<div class="form-group">
					<label for="fuel-sort">Sortierung (ohne Auswahl)</label>
					<select id="fuel-sort" bind:value={widget.config!.sort}>
						<option value="dist">Entfernung</option>
						<option value="price_e5">Günstigstes E5</option>
						<option value="price_e10">Günstigstes E10</option>
						<option value="price_diesel">Günstigstes Diesel</option>
					</select>
				</div>
				<div class="form-group">
					<label for="fuel-max">Max. Tankstellen (ohne Auswahl)</label>
					<input id="fuel-max" type="number" min="1" max="20" placeholder="5"
						value={widget.config!.max_stations ?? 5}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.max_stations = isNaN(v) ? 5 : v;
						}}
					/>
				</div>
				<div class="form-group">
					<label for="fuel-station-size">Darstellungsgröße</label>
					<select id="fuel-station-size"
						value={widget.config!.station_size ?? 'medium'}
						onchange={(e) => { widget.config!.station_size = (e.target as HTMLSelectElement).value; }}
					>
						<option value="small">Klein (kompakt, eine Zeile)</option>
						<option value="medium">Mittel (Name links, Preise rechts)</option>
						<option value="large">Groß (mehr Abstand, größere Schrift)</option>
					</select>
				</div>
				<div class="section-mini-title">Kraftstoffarten anzeigen</div>
				<div class="weather-toggles">
					<label class="toggle-row">
						<span>E5 (Super)</span>
						<label class="toggle">
							<input type="checkbox"
								checked={widget.config!.show_e5 !== false}
								onchange={(e) => { widget.config!.show_e5 = (e.target as HTMLInputElement).checked; }}
							/>
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>E10</span>
						<label class="toggle">
							<input type="checkbox"
								checked={widget.config!.show_e10 !== false}
								onchange={(e) => { widget.config!.show_e10 = (e.target as HTMLInputElement).checked; }}
							/>
							<span class="toggle-track"></span>
						</label>
					</label>
					<label class="toggle-row">
						<span>Diesel</span>
						<label class="toggle">
							<input type="checkbox"
								checked={widget.config!.show_diesel !== false}
								onchange={(e) => { widget.config!.show_diesel = (e.target as HTMLInputElement).checked; }}
							/>
							<span class="toggle-track"></span>
						</label>
					</label>
				</div>
			{/if}

			<!-- Grid config (bookmarks + services) -->
			{#if widget.type === 'bookmarks' || widget.type === 'services'}
				{@const defaultCols = widget.type === 'bookmarks' ? 1 : null}
				{@const layoutLabel = widget.type === 'bookmarks' ? 'bookmarks' : 'services'}
				<div class="form-group">
					<label for="{layoutLabel}-layout">Layout</label>
					<select id="{layoutLabel}-layout"
						value={widget.config!.columns ?? (widget.type === 'bookmarks' ? 1 : '')}
						onchange={(e) => {
							const raw = (e.target as HTMLSelectElement).value;
							widget.config!.columns = raw === '' ? undefined : parseInt(raw);
						}}
					>
						{#if widget.type === 'services'}
							<option value="">Automatisch (Flex-Umbruch)</option>
						{/if}
						<option value={1}>1 Spalte (Liste)</option>
						<option value={2}>2 Spalten</option>
						<option value={3}>3 Spalten</option>
						<option value={4}>4 Spalten</option>
						<option value={5}>5 Spalten</option>
						<option value={6}>6 Spalten</option>
						<option value={0}>Benutzerdefiniert</option>
					</select>
				</div>
				{#if (widget.config!.columns ?? (widget.type === 'bookmarks' ? 1 : null)) === 0}
					<div class="grid-custom-row">
						<div class="form-group">
							<label for="{layoutLabel}-grid-cols">Spalten</label>
							<input id="{layoutLabel}-grid-cols" type="number" min="1" max="12" placeholder="3"
								value={widget.config!.grid_cols ?? ''}
								oninput={(e) => {
									const v = parseInt((e.target as HTMLInputElement).value);
									widget.config!.grid_cols = isNaN(v) ? undefined : v;
								}}
							/>
						</div>
						<div class="form-group">
							<label for="{layoutLabel}-grid-min-w">Min. Breite (px)</label>
							<input id="{layoutLabel}-grid-min-w" type="number" min="40" max="400" placeholder="auto"
								value={widget.config!.grid_min_w ?? ''}
								oninput={(e) => {
									const v = parseInt((e.target as HTMLInputElement).value);
									widget.config!.grid_min_w = isNaN(v) ? undefined : v;
								}}
							/>
						</div>
					</div>
				{/if}
				<div class="form-group">
					<label for="{layoutLabel}-gap">Abstand zwischen Einträgen (px)</label>
					<input id="{layoutLabel}-gap" type="number" min="0" max="40"
						placeholder={widget.type === 'services' ? '8' : '2'}
						value={widget.config!.gap ?? ''}
						oninput={(e) => {
							const v = parseInt((e.target as HTMLInputElement).value);
							widget.config!.gap = isNaN(v) ? undefined : v;
						}}
					/>
				</div>
				<div class="form-group">
					<label for="{layoutLabel}-item-align">Ausrichtung der Einträge</label>
					<select id="{layoutLabel}-item-align"
						value={widget.config!.item_align ?? ''}
						onchange={(e) => {
							const v = (e.target as HTMLSelectElement).value;
							widget.config!.item_align = v || undefined;
						}}
					>
						<option value="">Standard</option>
						<option value="left">Links</option>
						<option value="center">Zentriert</option>
						<option value="right">Rechts</option>
					</select>
				</div>
			{/if}

			<!-- Items (services / bookmarks) -->
			{#if widget.type === 'services' || widget.type === 'bookmarks'}
				<div class="items-section">
					<div class="items-header">
						<span>Einträge</span>
						<button class="btn btn-ghost btn-sm" onclick={addItem}>+ Hinzufügen</button>
					</div>

					{#each widget.items ?? [] as item, i (itemKeys[i])}
						<div class="item-editor" class:is-draft={isDraft[i]}>
							<div class="item-header" onclick={() => toggle(i)} role="button" tabindex="0"
								onkeydown={(e) => e.key === 'Enter' && toggle(i)}>
								<span class="toggle-arrow">{expanded[i] ? '▾' : '▸'}</span>
								<span class="item-name-preview">{item.name || 'Neuer Eintrag'}</span>
								<div class="item-mini-controls">
									<button class="icon-btn-sm" onclick={(e) => { e.stopPropagation(); moveItem(i, -1); }} title="Nach oben">↑</button>
									<input
										class="pos-input"
										type="number"
										min="1"
										max={widget.items?.length ?? 1}
										bind:value={posInputs[i]}
										title="Position"
										onclick={(e) => e.stopPropagation()}
										onkeydown={(e) => { e.stopPropagation(); if (e.key === 'Enter') { moveItemTo(i, posInputs[i] - 1); (e.target as HTMLInputElement).blur(); } }}
										onblur={() => moveItemTo(i, posInputs[i] - 1)}
									/>
									<button class="icon-btn-sm" onclick={(e) => { e.stopPropagation(); moveItem(i, 1); }} title="Nach unten">↓</button>
									<button class="icon-btn-sm danger" onclick={(e) => { e.stopPropagation(); removeItem(i); }} title="Löschen">✕</button>
								</div>
							</div>

							{#if expanded[i]}
								<div class="item-fields">
									<div class="form-group">
										<label for="item-name-{i}">Name</label>
										<input id="item-name-{i}" type="text" bind:value={item.name} placeholder="Service Name" />
									</div>
									<div class="form-group">
										<label for="item-url-{i}">URL</label>
										<input id="item-url-{i}" type="url" bind:value={item.url} placeholder="https://..." />
									</div>
									<div class="form-group">
										<label for="item-icon-{i}">Icon (Name oder URL)</label>
										<input id="item-icon-{i}" type="text" bind:value={item.icon}
											placeholder="z.B. jellyfin, sonarr, oder https://..." />
										{#if item.icon && !item.icon.startsWith('http') && !item.icon.startsWith('/')}
											<div class="icon-preview">
												<img
													src="https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/{item.icon}.png"
													alt="Vorschau"
													width="24"
													onerror={(e) => ((e.target as HTMLImageElement).style.display = 'none')}
												/>
												<span class="hint">Dashboard Icons: <a href="https://github.com/walkxcode/dashboard-icons" target="_blank" rel="noopener">Liste ansehen</a></span>
											</div>
										{/if}
										<IconPicker
											value={item.icon}
											onchange={(url) => { if (widget.items) widget.items[i].icon = url; }}
										/>
									</div>
									<div class="form-group">
										<label for="item-description-{i}">Beschreibung</label>
										<input id="item-description-{i}" type="text" bind:value={item.description} placeholder="Optional" />
									</div>
									<div class="form-group">
										<label for="item-target-{i}">Öffnen in</label>
										<select id="item-target-{i}" bind:value={item.target}>
											<option value="_blank">Neuer Tab</option>
											<option value="_self">Gleiches Fenster</option>
										</select>
									</div>
									<div class="form-group">
										<label for="item-size-{i}">Größe</label>
										<select id="item-size-{i}" bind:value={item.size}>
											<option value={undefined}>Standard (mittel)</option>
											<option value="small">Klein (nur Icon)</option>
											<option value="medium">Mittel (Icon + Name)</option>
											<option value="large">Groß (Icon + Name + Beschreibung)</option>
										</select>
									</div>
									<div class="dim-row">
										<div class="form-group">
											<label for="item-width-{i}">Breite (px)</label>
											<input id="item-width-{i}" type="number" min="40" max="600" placeholder="Auto"
												value={item.item_width ?? ''}
												oninput={numInput(item.item_width, (v) => { if (widget.items) widget.items[i].item_width = v; })}
											/>
										</div>
										<div class="form-group">
											<label for="item-height-{i}">Höhe (px)</label>
											<input id="item-height-{i}" type="number" min="30" max="400" placeholder="Auto"
												value={item.item_height ?? ''}
												oninput={numInput(item.item_height, (v) => { if (widget.items) widget.items[i].item_height = v; })}
											/>
										</div>
									</div>
									<div class="dim-row">
										<div class="form-group">
											<label for="item-icon-size-{i}">Icongröße (px)</label>
											<input id="item-icon-size-{i}" type="number" min="12" max="200" placeholder="Standard"
												value={item.icon_size ?? ''}
												oninput={numInput(item.icon_size, (v) => { if (widget.items) widget.items[i].icon_size = v; })}
											/>
										</div>
									</div>
									<div class="dim-row">
										<div class="form-group">
											<label for="item-name-font-size-{i}">Schriftgröße (px)</label>
											<input id="item-name-font-size-{i}" type="number" min="8" max="48" placeholder="Standard"
												value={item.name_font_size ?? ''}
												oninput={numInput(item.name_font_size, (v) => { if (widget.items) widget.items[i].name_font_size = v; })}
											/>
										</div>
										<div class="form-group">
											<label for="item-font-family-{i}">Schriftart</label>
											<select
												id="item-font-family-{i}"
												value={item.font_family ?? ''}
												onchange={(e) => { if (widget.items) widget.items[i].font_family = (e.target as HTMLSelectElement).value || undefined; }}
											>
												<option value="">Standard (Theme)</option>
												{#each FONTS.slice(1) as f}
													<option value={f}>{f}</option>
												{/each}
											</select>
										</div>
									</div>
									<div class="form-group">
										<span class="label-text">Textausrichtung</span>
										<div class="align-btn-row">
											{#each ALIGN_OPTIONS as opt}
												<button
													type="button"
													class="badge-toggle-btn"
													class:active={item.text_align === opt.value || (!item.text_align && opt.value === 'left')}
													title={opt.title}
													onclick={() => {
														if (widget.items) widget.items[i].text_align =
															opt.value === 'left' ? undefined : opt.value as 'center' | 'right';
													}}
												>{opt.label}</button>
											{/each}
										</div>
									</div>
									<div class="section-mini-title">Farben (optional)</div>
									<div class="color-grid-2">
										<div class="form-group">
											<label for="item-text-color-{i}">Schrift</label>
											<div class="color-mini-row">
												<input id="item-text-color-{i}" type="color" class="color-picker-sm"
													value={cpVal(item.text_color, '#f1f5f9')}
													oninput={(e) => { if (widget.items) widget.items[i].text_color = (e.target as HTMLInputElement).value; }}
												/>
												<input type="text" placeholder="auto" value={item.text_color ?? ''} aria-label="Schriftfarbe als Text"
													oninput={(e) => { if (widget.items) widget.items[i].text_color = (e.target as HTMLInputElement).value || undefined; }}
												/>
											</div>
										</div>
										<div class="form-group">
											<label for="item-desc-color-{i}">Beschreibung</label>
											<div class="color-mini-row">
												<input id="item-desc-color-{i}" type="color" class="color-picker-sm"
													value={cpVal(item.desc_color, '#94a3b8')}
													oninput={(e) => { if (widget.items) widget.items[i].desc_color = (e.target as HTMLInputElement).value; }}
												/>
												<input type="text" placeholder="auto" value={item.desc_color ?? ''} aria-label="Beschreibungsschriftfarbe als Text"
													oninput={(e) => { if (widget.items) widget.items[i].desc_color = (e.target as HTMLInputElement).value || undefined; }}
												/>
											</div>
										</div>
										<div class="form-group">
											<label for="item-bg-color-{i}">Hintergrund</label>
											<div class="color-mini-row">
												<input id="item-bg-color-{i}" type="color" class="color-picker-sm"
													value={cpVal(item.bg_color, '#1e293b')}
													oninput={(e) => { if (widget.items) widget.items[i].bg_color = (e.target as HTMLInputElement).value; }}
												/>
												<input type="text" placeholder="auto" value={item.bg_color ?? ''} aria-label="Hintergrundfarbe als Text"
													oninput={(e) => { if (widget.items) widget.items[i].bg_color = (e.target as HTMLInputElement).value || undefined; }}
												/>
											</div>
										</div>
										<div class="form-group">
											<label for="item-border-color-{i}">Rahmen</label>
											<div class="color-mini-row">
												<input id="item-border-color-{i}" type="color" class="color-picker-sm"
													value={cpVal(item.border_color, '#ffffff')}
													oninput={(e) => { if (widget.items) widget.items[i].border_color = (e.target as HTMLInputElement).value; }}
												/>
												<input type="text" placeholder="auto" value={item.border_color ?? ''} aria-label="Rahmenfarbe als Text"
													oninput={(e) => { if (widget.items) widget.items[i].border_color = (e.target as HTMLInputElement).value || undefined; }}
												/>
											</div>
										</div>
										<div class="form-group" style="grid-column:1/-1">
											<label for="item-icon-bg-color-{i}">Icon-Hintergrund</label>
											<div class="color-mini-row">
												<input id="item-icon-bg-color-{i}" type="color" class="color-picker-sm"
													value={cpVal(item.icon_bg_color, '#334155')}
													oninput={(e) => { if (widget.items) widget.items[i].icon_bg_color = (e.target as HTMLInputElement).value; }}
												/>
												<input type="text" placeholder="auto" value={item.icon_bg_color ?? ''} aria-label="Icon-Hintergrundfarbe als Text"
													oninput={(e) => { if (widget.items) widget.items[i].icon_bg_color = (e.target as HTMLInputElement).value || undefined; }}
												/>
											</div>
										</div>
									</div>

									<button class="btn btn-ghost btn-sm clear-colors-btn" onclick={() => clearItemColors(i)}>
										Farben zurücksetzen
									</button>
									<button class="btn btn-ghost btn-sm apply-all-btn" onclick={() => applyToAll(i)}>
										Einstellungen auf alle übernehmen
									</button>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}

			<!-- Appearance -->
			<div class="section-divider"></div>
			<div class="section-title">Erscheinungsbild</div>

			<div class="form-group">
				<label for="widget-bg-color">Hintergrundfarbe</label>
				<div class="color-row">
					<input
						id="widget-bg-color"
						type="color"
						class="color-picker"
						value={widget.style?.background_color || '#1e293b'}
						oninput={(e) => setStyle('background_color', (e.target as HTMLInputElement).value)}
					/>
					<input
						type="text"
						value={widget.style?.background_color || ''}
						placeholder="Standard"
						aria-label="Hintergrundfarbe des Widgets als Text"
						oninput={(e) => setStyle('background_color', (e.target as HTMLInputElement).value)}
					/>
					{#if widget.style?.background_color}
						<button class="reset-btn" onclick={() => clearStyle('background_color')} title="Zurücksetzen">✕</button>
					{/if}
				</div>
			</div>

			<div class="form-group">
				<label for="widget-border-color">Rahmenfarbe</label>
				<div class="color-row">
					<input
						id="widget-border-color"
						type="color"
						class="color-picker"
						value={widget.style?.border_color || '#ffffff'}
						oninput={(e) => setStyle('border_color', (e.target as HTMLInputElement).value)}
					/>
					<input
						type="text"
						value={widget.style?.border_color || ''}
						placeholder="Standard"
						aria-label="Rahmenfarbe als Text"
						oninput={(e) => setStyle('border_color', (e.target as HTMLInputElement).value)}
					/>
					{#if widget.style?.border_color}
						<button class="reset-btn" onclick={() => clearStyle('border_color')} title="Zurücksetzen">✕</button>
					{/if}
				</div>
			</div>

			<div class="form-group">
				<label for="widget-title-color">Titelfarbe</label>
				<div class="color-row">
					<input id="widget-title-color" type="color" class="color-picker" value={widget.style?.title_color || '#f1f5f9'}
						oninput={(e) => setStyle('title_color', (e.target as HTMLInputElement).value)}
					/>
					<input type="text" value={widget.style?.title_color || ''} placeholder="Standard" aria-label="Titelfarbe als Text"
						oninput={(e) => setStyle('title_color', (e.target as HTMLInputElement).value)}
					/>
					{#if widget.style?.title_color}
						<button class="reset-btn" onclick={() => clearStyle('title_color')} title="Zurücksetzen">✕</button>
					{/if}
				</div>
			</div>

			<div class="form-group">
				<label for="widget-text-color">Schriftfarbe (Widget)</label>
				<div class="color-row">
					<input id="widget-text-color" type="color" class="color-picker" value={widget.style?.text_color || '#f1f5f9'}
						oninput={(e) => setStyle('text_color', (e.target as HTMLInputElement).value)}
					/>
					<input type="text" value={widget.style?.text_color || ''} placeholder="Standard" aria-label="Schriftfarbe als Text"
						oninput={(e) => setStyle('text_color', (e.target as HTMLInputElement).value)}
					/>
					{#if widget.style?.text_color}
						<button class="reset-btn" onclick={() => clearStyle('text_color')} title="Zurücksetzen">✕</button>
					{/if}
				</div>
			</div>

			<div class="form-group">
				<label for="widget-opacity">Deckkraft ({Math.round(parseFloat(widget.style?.opacity ?? '1') * 100)}%)</label>
				<input id="widget-opacity" type="range" min="0" max="100"
					value={Math.round(parseFloat(widget.style?.opacity ?? '1') * 100)}
					oninput={(e) => {
						const pct = parseInt((e.target as HTMLInputElement).value);
						if (pct >= 100) clearStyle('opacity');
						else setStyle('opacity', (pct / 100).toFixed(2));
					}}
				/>
			</div>

			<div class="form-group">
				<button class="btn btn-ghost btn-sm clear-all-colors-btn" onclick={clearAllColors}>
					Alle Farben zurücksetzen
				</button>
			</div>

			<div class="form-group">
				<span class="label-text">Hintergrundbild</span>
				<WallpaperPicker
					value={widget.style?.background_image ?? ''}
					onchange={(url) => {
						if (url) setStyle('background_image', url);
						else clearStyle('background_image');
					}}
				/>
				<input
					type="text"
					style="margin-top:6px"
					value={widget.style?.background_image ?? ''}
					placeholder="Oder externe URL"
					aria-label="Externe Hintergrundbild-URL"
					oninput={(e) => {
						const v = (e.target as HTMLInputElement).value;
						if (v) setStyle('background_image', v);
						else clearStyle('background_image');
					}}
				/>
			</div>
		</div>

		<div class="panel-footer">
			<button class="btn btn-ghost" onclick={discard}>Verwerfen</button>
			<button class="btn btn-primary" onclick={save}>Speichern</button>
		</div>
	</div>
</div>

<style>
	.items-section {
		margin-top: 8px;
	}

	.items-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 12px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.item-editor {
		border: 1px solid var(--border-subtle);
		border-radius: 10px;
		margin-bottom: 6px;
		overflow: hidden;
	}

	.item-editor.is-draft {
		border-color: rgba(129,140,248,0.35);
	}

	.item-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 9px 10px;
		cursor: pointer;
		user-select: none;
		background: var(--item-hover);
		transition: background 0.12s;
	}

	.item-header:hover {
		background: var(--item-hover);
		filter: brightness(1.4);
	}

	.toggle-arrow {
		font-size: 10px;
		color: var(--text-muted);
		flex-shrink: 0;
		width: 12px;
	}

	.item-name-preview {
		flex: 1;
		font-size: 12px;
		font-weight: 500;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.item-mini-controls {
		display: flex;
		gap: 4px;
		flex-shrink: 0;
	}

	.item-fields {
		padding: 12px;
		border-top: 1px solid var(--border-subtle);
	}

	.icon-btn-sm {
		width: 24px;
		height: 24px;
		background: rgba(255,255,255,0.07);
		border: 1px solid rgba(255,255,255,0.1);
		border-radius: 5px;
		cursor: pointer;
		color: var(--text);
		font-size: 11px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.icon-btn-sm.danger {
		border-color: rgba(239,68,68,0.3);
		color: #f87171;
	}

	.icon-btn-sm.danger:hover {
		background: rgba(239,68,68,0.2);
	}

	.pos-input {
		width: 36px;
		height: 24px;
		background: rgba(255,255,255,0.07);
		border: 1px solid rgba(255,255,255,0.1);
		border-radius: 5px;
		color: var(--text);
		font-size: 11px;
		text-align: center;
		padding: 0 2px;
		-moz-appearance: textfield;
	}
	.pos-input::-webkit-inner-spin-button,
	.pos-input::-webkit-outer-spin-button {
		display: none;
	}
	.pos-input:focus {
		outline: none;
		border-color: rgba(255,255,255,0.3);
	}

	.btn-sm {
		padding: 4px 10px;
		font-size: 12px;
	}

	.dim-row,
	.grid-custom-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
	}

	.apply-all-btn {
		width: 100%;
		margin-top: 4px;
		font-size: 11px;
		justify-content: center;
	}

	.clear-colors-btn {
		width: 100%;
		margin-top: 2px;
		font-size: 11px;
		justify-content: center;
		color: #f87171;
		border-color: rgba(239, 68, 68, 0.25);
	}

	.clear-colors-btn:hover {
		background: rgba(239, 68, 68, 0.12);
		color: #fca5a5;
	}

	.clear-all-colors-btn {
		width: 100%;
		font-size: 12px;
		justify-content: center;
		color: #f87171;
		border-color: rgba(239, 68, 68, 0.25);
	}

	.clear-all-colors-btn:hover {
		background: rgba(239, 68, 68, 0.12);
		color: #fca5a5;
	}

	.section-mini-title {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.07em;
		text-transform: uppercase;
		color: var(--text-muted);
		margin: 12px 0 6px;
	}

	.color-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2px 8px;
	}

	.color-mini-row {
		display: flex;
		gap: 4px;
		align-items: center;
	}

	.color-picker-sm {
		width: 28px !important;
		height: 28px !important;
		padding: 2px !important;
		flex-shrink: 0;
		cursor: pointer;
		border-radius: 4px !important;
	}

	.color-mini-row input[type="text"] {
		font-size: 11px;
		padding: 5px 6px;
	}

	.regions-dropdown { width: 100%; }

	.regions-dropdown[open] > .regions-trigger { border-radius: 8px 8px 0 0; }

	.regions-trigger {
		width: 100%;
		box-sizing: border-box;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: rgba(255, 255, 255, 0.07);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		cursor: pointer;
		font-size: 14px;
		font-family: var(--font);
		color: var(--text);
		list-style: none;
		user-select: none;
	}

	.regions-trigger::-webkit-details-marker { display: none; }
	.regions-trigger::after { content: '▾'; font-size: 10px; color: var(--text-muted); margin-left: 6px; }
	.regions-dropdown[open] .regions-trigger::after { content: '▴'; }

	.regions-list {
		position: relative;
		background: var(--surface);
		border: 1px solid var(--border-subtle);
		border-top: none;
		border-radius: 0 0 8px 8px;
		max-height: 220px;
		overflow-y: auto;
		padding: 4px 0;
		z-index: 10;
	}

	.regions-divider {
		height: 1px;
		background: rgba(255, 255, 255, 0.08);
		margin: 4px 0;
	}

	.region-option {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 5px 12px;
		font-size: 12px;
		color: var(--text);
		cursor: pointer;
	}

	.region-option:hover { background: rgba(255, 255, 255, 0.06); }

	.region-option input[type="checkbox"] {
		accent-color: var(--primary);
		cursor: pointer;
		flex-shrink: 0;
	}

	.icon-preview {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 4px;
		font-size: 11px;
		color: var(--text-muted);
	}

	.icon-preview a {
		color: var(--primary);
	}

	/* Fuel station picker */
	.fuel-search-btn {
		width: 100%;
		justify-content: center;
		margin-bottom: 6px;
	}

	.fuel-results {
		border: 1px solid rgba(255,255,255,0.1);
		border-radius: 8px;
		overflow: hidden;
		max-height: 260px;
		overflow-y: auto;
	}

	.fuel-result-row {
		display: grid;
		grid-template-columns: auto 1fr auto;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		width: 100%;
		cursor: pointer;
		border-bottom: 1px solid rgba(255,255,255,0.05);
		transition: background 0.12s;
		box-sizing: border-box;
	}

	.fuel-result-row:last-child { border-bottom: none; }
	.fuel-result-row:hover { background: rgba(255,255,255,0.05); }
	.fuel-result-row.selected { background: rgba(129,140,248,0.1); }

	.fuel-result-row input[type="checkbox"] {
		accent-color: var(--primary);
		cursor: pointer;
	}

	.fuel-result-info {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.fuel-result-name {
		font-size: 12px;
		font-weight: 600;
		color: var(--text);
	}

	.fuel-result-addr {
		font-size: 11px;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.fuel-result-dist {
		font-size: 11px;
		color: var(--text-muted);
		flex-shrink: 0;
	}

	.fuel-selected-list {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.fuel-sel-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 10px;
		background: rgba(255,255,255,0.03);
		border: 1px solid rgba(255,255,255,0.08);
		border-radius: 8px;
	}

	.fuel-sel-meta {
		display: flex;
		flex-direction: column;
		min-width: 0;
		flex: 1;
	}

	.fuel-sel-name {
		font-size: 12px;
		font-weight: 600;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.fuel-sel-addr {
		font-size: 10px;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.fuel-sel-caption {
		flex: 1;
		min-width: 80px;
		font-size: 12px !important;
		padding: 5px 8px !important;
	}

	.fuel-sel-actions {
		display: flex;
		gap: 3px;
		flex-shrink: 0;
	}

	.weather-toggles {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin-bottom: 4px;
	}

	.toggle-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 13px;
		color: var(--text);
		cursor: pointer;
		padding: 2px 0;
	}

	.toggle {
		display: flex;
		align-items: center;
		cursor: pointer;
	}

	.toggle input { display: none; }

	.toggle-track {
		width: 40px;
		height: 22px;
		background: rgba(255,255,255,0.15);
		border-radius: 11px;
		position: relative;
		transition: background 0.2s;
	}

	.toggle-track::after {
		content: '';
		position: absolute;
		left: 3px;
		top: 3px;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: #fff;
		transition: transform 0.2s;
	}

	.toggle input:checked ~ .toggle-track {
		background: var(--primary);
	}

	.toggle input:checked ~ .toggle-track::after {
		transform: translateX(18px);
	}

	.section-divider {
		border-top: 1px solid var(--border-subtle);
		margin: 20px 0 16px;
	}

	.section-title {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--text-muted);
		margin-bottom: 14px;
	}

	.color-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.color-picker {
		width: 36px;
		height: 34px;
		padding: 2px;
		flex-shrink: 0;
		cursor: pointer;
		border-radius: 6px;
	}

	.reset-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 12px;
		cursor: pointer;
		flex-shrink: 0;
		padding: 4px;
		line-height: 1;
	}

	.reset-btn:hover {
		color: #f87171;
	}

	.panel-footer {
		padding: 16px 24px;
		border-top: 1px solid rgba(255,255,255,0.07);
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 18px;
		cursor: pointer;
	}

	.hint {
		font-size: 10px;
		color: var(--text-muted);
		margin-top: 3px;
		display: block;
	}

	.badge-controls-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
		align-items: end;
	}

	.badge-fs-group input {
		width: 100%;
	}

	.badge-toggle-row,
	.align-btn-row {
		display: flex;
		gap: 4px;
	}

	.badge-toggle-btn {
		width: 32px;
		height: 32px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 6px;
		color: var(--text-muted);
		cursor: pointer;
		font-size: 13px;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background 0.12s, color 0.12s, border-color 0.12s;
		flex-shrink: 0;
	}

	.badge-toggle-btn.active {
		background: rgba(129, 140, 248, 0.2);
		border-color: rgba(129, 140, 248, 0.5);
		color: var(--primary);
	}

	.badge-toggle-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		color: var(--text);
	}

	/* RSS preset buttons */
	.rss-presets {
		display: flex;
		flex-wrap: wrap;
		gap: 5px;
		margin-bottom: 10px;
	}

	.rss-preset-btn {
		font-size: 11px;
		padding: 4px 9px;
		border-radius: 20px;
		border: 1px solid var(--border-subtle);
		background: var(--item-hover);
		color: var(--text-muted);
		cursor: pointer;
		transition: background 0.12s, color 0.12s, border-color 0.12s;
		white-space: nowrap;
	}

	.rss-preset-btn:not(:disabled):hover {
		background: var(--primary);
		color: #fff;
		border-color: var(--primary);
	}

	.rss-preset-btn.added,
	.rss-preset-btn:disabled {
		background: rgba(129, 140, 248, 0.15);
		color: var(--primary);
		border-color: var(--primary);
		cursor: default;
		opacity: 0.75;
	}

	.rss-source-row {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 6px;
		align-items: start;
		padding: 10px;
		background: var(--item-hover);
		border: 1px solid var(--border-subtle);
		border-radius: 8px;
		margin-bottom: 6px;
	}

	/* Calendar multi-source */
	.cal-source-row {
		display: grid;
		grid-template-columns: 28px 1fr auto;
		gap: 6px;
		align-items: start;
		padding: 10px;
		background: var(--item-hover);
		border: 1px solid var(--border-subtle);
		border-radius: 8px;
		margin-bottom: 6px;
	}

	.cal-color-wrap {
		display: flex;
		align-items: flex-start;
		padding-top: 6px;
	}

	.cal-color-picker {
		width: 28px !important;
		height: 28px !important;
		padding: 2px !important;
		border-radius: 50% !important;
		cursor: pointer;
		flex-shrink: 0;
	}

	.cal-source-fields {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
	}

	.cal-source-fields input {
		width: 100%;
		font-size: 12px;
		padding: 6px 8px;
	}

	.cal-remove-btn {
		margin-top: 6px;
	}

	.cal-add-btn {
		width: 100%;
		justify-content: center;
		font-size: 12px;
	}
</style>
