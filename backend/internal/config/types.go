package config

type Dashboard struct {
	Theme   Theme    `yaml:"theme"   json:"theme"`
	Widgets []Widget `yaml:"widgets" json:"widgets"`
}

type Theme struct {
	BackgroundImage string `yaml:"background_image" json:"background_image"`
	BackgroundColor string `yaml:"background_color" json:"background_color"`
	CardColor       string `yaml:"card_color"       json:"card_color"`
	PrimaryColor    string `yaml:"primary_color"    json:"primary_color"`
	AccentColor     string `yaml:"accent_color"     json:"accent_color"`
	TextColor       string `yaml:"text_color"       json:"text_color"`
	GlassEffect     bool   `yaml:"glass_effect"     json:"glass_effect"`
	BorderRadius    int    `yaml:"border_radius"    json:"border_radius"`
	Font            string `yaml:"font"             json:"font"`
}

type Widget struct {
	ID       string                 `yaml:"id"                json:"id"`
	Type     string                 `yaml:"type"              json:"type"`
	Title    string                 `yaml:"title"             json:"title"`
	Position Position               `yaml:"position"          json:"position"`
	Items    []WidgetItem           `yaml:"items,omitempty"   json:"items,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty"  json:"config,omitempty"`
	Style    map[string]string      `yaml:"style,omitempty"   json:"style,omitempty"`
}

type Position struct {
	X int `yaml:"x" json:"x"`
	Y int `yaml:"y" json:"y"`
	W int `yaml:"w" json:"w"`
	H int `yaml:"h" json:"h"`
}

type WidgetItem struct {
	Name        string `yaml:"name"                  json:"name"`
	URL         string `yaml:"url"                   json:"url"`
	Icon        string `yaml:"icon"                  json:"icon"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Tag         string `yaml:"tag,omitempty"         json:"tag,omitempty"`
	Target      string `yaml:"target,omitempty"      json:"target,omitempty"`
	Size         string `yaml:"size,omitempty"           json:"size,omitempty"`
	ItemWidth    int    `yaml:"item_width,omitempty"     json:"item_width,omitempty"`
	ItemHeight   int    `yaml:"item_height,omitempty"    json:"item_height,omitempty"`
	NameFontSize int    `yaml:"name_font_size,omitempty" json:"name_font_size,omitempty"`
	DescFontSize int    `yaml:"desc_font_size,omitempty" json:"desc_font_size,omitempty"`
	FontFamily   string `yaml:"font_family,omitempty"    json:"font_family,omitempty"`
	TextAlign    string `yaml:"text_align,omitempty"     json:"text_align,omitempty"`
	TextColor    string `yaml:"text_color,omitempty"     json:"text_color,omitempty"`
	DescColor    string `yaml:"desc_color,omitempty"     json:"desc_color,omitempty"`
	BgColor      string `yaml:"bg_color,omitempty"       json:"bg_color,omitempty"`
	BorderColor  string `yaml:"border_color,omitempty"   json:"border_color,omitempty"`
	IconBgColor  string `yaml:"icon_bg_color,omitempty"  json:"icon_bg_color,omitempty"`
}

func DefaultDashboard() *Dashboard {
	return &Dashboard{
		Theme: Theme{
			BackgroundColor: "#0f172a",
			CardColor:       "rgba(30, 41, 59, 0.85)",
			PrimaryColor:    "#818cf8",
			AccentColor:     "#fb923c",
			TextColor:       "#f1f5f9",
			GlassEffect:     true,
			BorderRadius:    14,
			Font:            "Inter",
		},
		Widgets: []Widget{
			{
				ID:       "clock-1",
				Type:     "clock",
				Title:    "",
				Position: Position{X: 0, Y: 0, W: 3, H: 3},
				Config: map[string]interface{}{
					"format":    "24h",
					"show_date": true,
					"timezone":  "Europe/Berlin",
				},
			},
			{
				ID:       "services-1",
				Type:     "services",
				Title:    "Services",
				Position: Position{X: 3, Y: 0, W: 9, H: 4},
				Items: []WidgetItem{
					{Name: "Portainer", URL: "http://localhost:9000", Icon: "portainer", Description: "Container Management"},
					{Name: "Nextcloud", URL: "http://localhost:8080", Icon: "nextcloud", Description: "Datei-Speicher"},
					{Name: "Home Assistant", URL: "http://localhost:8123", Icon: "home-assistant", Description: "Smart Home"},
					{Name: "Jellyfin", URL: "http://localhost:8096", Icon: "jellyfin", Description: "Media Server"},
					{Name: "Grafana", URL: "http://localhost:3000", Icon: "grafana", Description: "Monitoring"},
					{Name: "Gitea", URL: "http://localhost:3001", Icon: "gitea", Description: "Git Server"},
				},
			},
			{
				ID:       "bookmarks-1",
				Type:     "bookmarks",
				Title:    "Links",
				Position: Position{X: 0, Y: 3, W: 3, H: 4},
				Config: map[string]interface{}{
					"columns": 1,
					"gap":     4,
				},
				Items: []WidgetItem{
					{Name: "GitHub", URL: "https://github.com", Icon: "github", Description: "Code Hosting"},
					{Name: "Google", URL: "https://google.com", Icon: "google", Description: "Suche"},
					{Name: "Reddit", URL: "https://reddit.com", Icon: "reddit", Description: "Community"},
					{Name: "YouTube", URL: "https://youtube.com", Icon: "youtube", Description: "Videos"},
				},
			},
			{
				ID:       "weather-1",
				Type:     "weather",
				Title:    "Wetter",
				Position: Position{X: 3, Y: 4, W: 3, H: 3},
				Config: map[string]interface{}{
					"city":  "Berlin",
					"units": "celsius",
				},
			},
		},
	}
}
