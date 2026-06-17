import type { Dashboard, Theme, Widget, WeatherData, Wallpaper, Icon, NewsResponse, DockerData, GarbageData, FuelData, FuelPricesData, CalendarData, RssItem, RssSource } from './types';

const base = '';

export class AuthError extends Error {
	constructor() { super('Unauthorized'); }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(base + path, {
		headers: { 'Content-Type': 'application/json' },
		...init,
	});
	if (res.status === 401) throw new AuthError();
	if (!res.ok) {
		let msg = `${res.status} ${res.statusText}`;
		try { const b = await res.json(); if (b?.error) msg = b.error; } catch { /* ignore */ }
		throw new Error(msg);
	}
	if (res.status === 204) return undefined as T;
	return res.json();
}

export const api = {
	getAuthInfo: () => request<{ enabled: boolean }>('/api/auth'),
	login: (password: string) =>
		request<{ ok: boolean }>('/api/login', { method: 'POST', body: JSON.stringify({ password }) }),
	logout: () => request<void>('/api/logout', { method: 'POST' }),

	getConfig: () => request<Dashboard>('/api/config'),
	updateTheme: (theme: Theme) =>
		request<Theme>('/api/config/theme', { method: 'PUT', body: JSON.stringify(theme) }),

	getWidgets: () => request<Widget[]>('/api/widgets'),
	createWidget: (w: Widget) =>
		request<Widget>('/api/widgets', { method: 'POST', body: JSON.stringify(w) }),
	updateWidget: (w: Widget) =>
		request<Widget>(`/api/widgets/${w.id}`, { method: 'PUT', body: JSON.stringify(w) }),
	deleteWidget: (id: string) =>
		request<void>(`/api/widgets/${id}`, { method: 'DELETE' }),
	updateLayout: (items: Array<{ id: string; x: number; y: number; w: number; h: number }>) =>
		request<void>('/api/widgets/layout', { method: 'PUT', body: JSON.stringify(items) }),

	getWeather: (city: string, units = 'celsius', forecastDays = 7) =>
		request<WeatherData>(`/api/weather?city=${encodeURIComponent(city)}&units=${units}&forecast_days=${forecastDays}`),

	getWallpapers: () => request<Wallpaper[]>('/api/wallpapers'),
	uploadWallpaper: async (file: File): Promise<Wallpaper> => {
		const fd = new FormData();
		fd.append('file', file);
		const res = await fetch('/api/wallpapers', { method: 'POST', body: fd });
		if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
		return res.json();
	},
	deleteWallpaper: (name: string) =>
		request<void>(`/api/wallpapers/${encodeURIComponent(name)}`, { method: 'DELETE' }),

	getIcons: () => request<Icon[]>('/api/icons'),
	uploadIcon: async (file: File): Promise<Icon> => {
		const fd = new FormData();
		fd.append('file', file);
		const res = await fetch('/api/icons', { method: 'POST', body: fd });
		if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
		return res.json();
	},
	deleteIcon: (name: string) =>
		request<void>(`/api/icons/${encodeURIComponent(name)}`, { method: 'DELETE' }),

	getDocker: (endpoint: string, showStopped = false, max = 10) =>
		request<DockerData>(`/api/docker?endpoint=${encodeURIComponent(endpoint)}&show_stopped=${showStopped}&max=${max}`),

	getGarbage: (source: string, daysAhead = 30, maxItems = 20) =>
		request<GarbageData>(`/api/garbage?source=${encodeURIComponent(source)}&days=${daysAhead}&max=${maxItems}`),

	getCalendar: (source: string, daysAhead = 30, maxItems = 20) =>
		request<CalendarData>(`/api/calendar?source=${encodeURIComponent(source)}&days=${daysAhead}&max=${maxItems}`),

	getFuel: (params: { apiKey: string; location?: string; lat?: number; lng?: number; radius?: number; sort?: string; max?: number }) => {
		const q = new URLSearchParams();
		q.set('api_key', params.apiKey);
		if (params.location) q.set('location', params.location);
		if (params.lat) q.set('lat', String(params.lat));
		if (params.lng) q.set('lng', String(params.lng));
		if (params.radius) q.set('radius', String(params.radius));
		if (params.sort) q.set('sort', params.sort);
		if (params.max) q.set('max', String(params.max));
		return request<FuelData>(`/api/fuel?${q.toString()}`);
	},

	getFuelPrices: (apiKey: string, ids: string[]) =>
		request<FuelPricesData>(`/api/fuel/prices?api_key=${encodeURIComponent(apiKey)}&ids=${ids.map(encodeURIComponent).join(',')}`),

	getNews: (params?: { regions?: number[]; ressort?: string; pageSize?: number }) => {
		const q = new URLSearchParams();
		if (params?.regions?.length) q.set('regions', params.regions.join(','));
		if (params?.ressort) q.set('ressort', params.ressort);
		if (params?.pageSize) q.set('pageSize', String(params.pageSize));
		const qs = q.toString();
		return request<NewsResponse>(`/api/news${qs ? '?' + qs : ''}`);
	},

	getRss: (sources: RssSource[], maxItems?: number) => {
		const q = new URLSearchParams();
		for (const s of sources) {
			q.append('source', s.url);
			q.append('name', s.name);
		}
		if (maxItems) q.set('max_items', String(maxItems));
		return request<{ items: RssItem[] }>(`/api/rss?${q.toString()}`);
	},
};
