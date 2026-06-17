export interface Theme {
	background_image: string;
	background_color: string;
	card_color: string;
	primary_color: string;
	accent_color: string;
	text_color: string;
	glass_effect: boolean;
	border_radius: number;
	font: string;
}

export interface Position {
	x: number;
	y: number;
	w: number;
	h: number;
}

export interface WidgetItem {
	name: string;
	url: string;
	icon: string;
	description?: string;
	tag?: string;
	target?: string;
	size?: 'small' | 'medium' | 'large';
	item_width?: number;
	item_height?: number;
	icon_size?: number;
	name_font_size?: number;
	desc_font_size?: number;
	font_family?: string;
	text_align?: 'left' | 'center' | 'right';
	text_color?: string;
	desc_color?: string;
	bg_color?: string;
	border_color?: string;
	icon_bg_color?: string;
}

export type WidgetType = 'services' | 'weather' | 'clock' | 'bookmarks' | 'news' | 'docker' | 'garbage' | 'fuel' | 'calendar';

export interface RssSource {
	name: string;
	url: string;
}

export interface RssItem {
	title: string;
	link: string;
	description: string;
	pub_date: string;
	source_name: string;
	source_url: string;
}

export interface WidgetStyle {
	background_color?: string;
	background_image?: string;
	border_color?: string;
	title_color?: string;
	text_color?: string;
	opacity?: string;
	topline_color?: string;
	topline_bg?: string;
	topline_font_size?: string;
	topline_bold?: string;
	topline_italic?: string;
	topline_border?: string;
}

export interface Widget {
	id: string;
	type: WidgetType;
	title: string;
	position: Position;
	items?: WidgetItem[];
	config?: Record<string, any>;
	style?: WidgetStyle;
}

export interface Wallpaper {
	name: string;
	url: string;
}

export interface Icon {
	name: string;
	url: string;
}

export interface Dashboard {
	theme: Theme;
	widgets: Widget[];
}

export interface WeatherData {
	city: string;
	country: string;
	temperature: number;
	feels_like: number;
	humidity: number;
	wind_speed: number;
	wind_direction: number;
	wind_gusts: number;
	weather_code: number;
	icon: string;
	description: string;
	uv_index: number;
	pressure: number;
	visibility: number;
	dew_point: number;
	cloud_cover: number;
	precipitation: number;
	forecast: ForecastDay[];
	units: string;
}

export interface ForecastDay {
	date: string;
	weather_code: number;
	icon: string;
	temp_max: number;
	temp_min: number;
	precip_prob: number;
	sunrise: string;
	sunset: string;
	uv_index_max: number;
	precip_sum: number;
	wind_max: number;
	wind_dominant: number;
}

export interface DockerData {
	running: number;
	stopped: number;
	total: number;
	containers: ContainerInfo[];
}

export interface ContainerInfo {
	id: string;
	name: string;
	image: string;
	state: string;
	status: string;
	created: number;
	ports: string[];
}

export interface GarbageData {
	events: GarbageEvent[];
	next: GarbageEvent | null;
}

export interface GarbageEvent {
	date: string;
	summary: string;
	icon: string;
	days_until: number;
}

export interface FuelData {
	stations: FuelStation[];
	location: string;
}

export interface FuelStation {
	id: string;
	name: string;
	brand: string;
	street: string;
	city: string;
	dist: number;
	e5: number;
	e10: number;
	diesel: number;
	is_open: boolean;
}

export interface FuelPricesData {
	prices: Record<string, FuelPrice>;
}

export interface FuelPrice {
	e5: number;
	e10: number;
	diesel: number;
	is_open: boolean;
}

export interface SelectedStation {
	id: string;
	caption: string;
	name: string;
	brand: string;
	street: string;
	city: string;
}

export interface CalendarSource {
	name: string;
	url: string;
	color?: string;
}

export interface CalendarData {
	events: CalendarEvent[];
}

export interface CalendarEvent {
	date: string;
	time: string;
	end_date?: string;
	end_time?: string;
	all_day: boolean;
	summary: string;
	location?: string;
	days_until: number;
}

export interface NewsArticle {
	sophoraId: string;
	type?: 'story' | 'video' | string;
	title: string;
	teaserText?: string;
	topline?: string;
	firstSentence?: string;
	date: string;
	detailsweb?: string;
	shareURL?: string;
	url?: string;
	streams?: {
		h264s?: string;
		h264m?: string;
		h264xl?: string;
		adaptivestreaming?: string;
	};
	ressort?: string;
	regionId?: number;
	tags?: Array<{ tag: string }>;
	teaserImage?: {
		imageVariants?: Record<string, { src?: string }>;
	};
}

export interface NewsResponse {
	news: NewsArticle[];
	nextPage?: string;
}

export const WIDGET_TEMPLATES: Record<WidgetType, Omit<Widget, 'id' | 'position'>> = {
	services: {
		type: 'services',
		title: 'Services',
		items: [
			{ name: 'Beispiel', url: 'https://example.com', icon: 'globe', description: 'Service' }
		]
	},
	weather: {
		type: 'weather',
		title: 'Wetter',
		config: { city: 'Berlin', units: 'celsius' }
	},
	clock: {
		type: 'clock',
		title: '',
		config: { format: '24h', show_date: true, timezone: 'Europe/Berlin' }
	},
	bookmarks: {
		type: 'bookmarks',
		title: 'Links',
		config: { columns: 1, gap: 2 },
		items: [
			{ name: 'GitHub', url: 'https://github.com', icon: 'github', description: '' }
		]
	},
	news: {
		type: 'news',
		title: 'Nachrichten',
		config: {
			mode: 'rss',
			rss_sources: [] as RssSource[],
			max_items: 20,
			refresh_interval: 10,
			ressort: '',
			regions: [],
			page_size: 10,
		}
	},
	docker: {
		type: 'docker',
		title: 'Docker',
		config: { endpoint: 'unix:///var/run/docker.sock', show_stopped: false, max_items: 10 }
	},
	garbage: {
		type: 'garbage',
		title: 'Müllkalender',
		config: { source: '', days_ahead: 30, max_items: 10 }
	},
	fuel: {
		type: 'fuel',
		title: 'Benzinpreise',
		config: { location: '', radius: 5, api_key: '', max_stations: 5, sort: 'dist', show_e5: true, show_e10: true, show_diesel: true }
	},
	calendar: {
		type: 'calendar',
		title: 'Kalender',
		config: { sources: [] as CalendarSource[], days_ahead: 30, max_items: 20 }
	}
};

export const DEFAULT_SIZES: Record<WidgetType, { w: number; h: number }> = {
	services:  { w: 4, h: 4 },
	weather:   { w: 3, h: 3 },
	clock:     { w: 2, h: 2 },
	bookmarks: { w: 3, h: 4 },
	news:      { w: 4, h: 5 },
	docker:    { w: 3, h: 4 },
	garbage:   { w: 3, h: 3 },
	fuel:      { w: 3, h: 4 },
	calendar:  { w: 3, h: 4 },
};
