import { api, AuthError } from '$lib/api';
import type { Dashboard, Theme, Widget } from '$lib/types';

function createDashboardStore() {
	let config = $state<Dashboard | null>(null);
	// null = not yet checked, true = logged in, false = needs login
	let authenticated = $state<boolean | null>(null);
	let editMode = $state(false);
	let settingsOpen = $state(false);
	let addWidgetOpen = $state(false);
	let editingWidget = $state<Widget | null>(null);
	let colorMode = $state<'dark' | 'light' | 'custom'>('dark');
	let ws: WebSocket | null = null;

	async function load() {
		try {
			config = await api.getConfig();
			authenticated = true;
			connectWS();
		} catch (e) {
			if (e instanceof AuthError) {
				authenticated = false;
			}
		}
	}

	async function login(password: string): Promise<string | null> {
		try {
			await api.login(password);
			await load();
			return null;
		} catch (e) {
			if (e instanceof AuthError) return 'Falsches Passwort';
			return e instanceof Error ? e.message : 'Fehler';
		}
	}

	async function logout() {
		await api.logout().catch(() => {});
		authenticated = false;
		config = null;
		editMode = false;
		if (ws) { ws.close(); ws = null; }
	}

	function connectWS() {
		const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
		ws = new WebSocket(`${proto}//${location.host}/ws`);
		ws.onmessage = (e) => {
			const msg = JSON.parse(e.data);
			if (msg.type === 'config') {
				config = msg.payload;
			}
		};
		ws.onclose = () => {
			setTimeout(connectWS, 2000);
		};
	}

	async function updateTheme(theme: Theme) {
		if (!config) return;
		config.theme = theme;
		await api.updateTheme(theme);
	}

	async function addWidget(widget: Widget) {
		await api.createWidget(widget);
		if (config) {
			config.widgets = [...config.widgets, widget];
		}
	}

	async function updateWidget(widget: Widget) {
		await api.updateWidget(widget);
		if (config) {
			config.widgets = config.widgets.map((w) => (w.id === widget.id ? widget : w));
		}
		editingWidget = null;
	}

	async function deleteWidget(id: string) {
		await api.deleteWidget(id);
		if (config) {
			config.widgets = config.widgets.filter((w) => w.id !== id);
		}
	}

	async function saveLayout(items: Array<{ id: string; x: number; y: number; w: number; h: number }>) {
		if (!config) return;
		for (const item of items) {
			const w = config.widgets.find((w) => w.id === item.id);
			if (w) {
				w.position.x = item.x;
				w.position.y = item.y;
				w.position.w = item.w;
				w.position.h = item.h;
			}
		}
		await api.updateLayout(items);
	}

	return {
		get config() { return config; },
		get authenticated() { return authenticated; },
		get editMode() { return editMode; },
		set editMode(v) { editMode = v; },
		get settingsOpen() { return settingsOpen; },
		set settingsOpen(v) { settingsOpen = v; },
		get addWidgetOpen() { return addWidgetOpen; },
		set addWidgetOpen(v) { addWidgetOpen = v; },
		get editingWidget() { return editingWidget; },
		set editingWidget(v) { editingWidget = v; },
		get colorMode() { return colorMode; },
		set colorMode(v: 'dark' | 'light' | 'custom') { colorMode = v; },
		load,
		login,
		logout,
		updateTheme,
		addWidget,
		updateWidget,
		deleteWidget,
		saveLayout,
	};
}

export const dashboard = createDashboardStore();
