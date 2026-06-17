package providers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type RssItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	PubDate     string `json:"pub_date"`
	SourceName  string `json:"source_name"`
	SourceURL   string `json:"source_url"`
}

// RSS 2.0
type rss20Feed struct {
	Channel rss20Channel `xml:"channel"`
}
type rss20Channel struct {
	Title string      `xml:"title"`
	Items []rss20Item `xml:"item"`
}
type rss20Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Atom
type atomFeed struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}
type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
}
type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

var rssCache struct {
	sync.Mutex
	entries map[string]rssCacheEntry
}

type rssCacheEntry struct {
	items   []RssItem
	expires time.Time
}

func init() {
	rssCache.entries = make(map[string]rssCacheEntry)
}

func FetchRss(sourceURL, sourceName string) ([]RssItem, error) {
	if err := ValidateURL(sourceURL); err != nil {
		return nil, fmt.Errorf("rss source blocked: %w", err)
	}

	rssCache.Lock()
	if e, ok := rssCache.entries[sourceURL]; ok && time.Now().Before(e.expires) {
		rssCache.Unlock()
		return e.items, nil
	}
	rssCache.Unlock()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rss request %s: %w", sourceURL, err)
	}
	req.Header.Set("User-Agent", "NavioDeck/1.0 (RSS Reader)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss fetch %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MB limit
	if err != nil {
		return nil, fmt.Errorf("rss read %s: %w", sourceURL, err)
	}

	items, chanTitle, err := parseRssBody(body)
	if err != nil {
		return nil, fmt.Errorf("rss parse %s: %w", sourceURL, err)
	}

	name := sourceName
	if name == "" {
		name = chanTitle
	}
	for i := range items {
		items[i].SourceName = name
		items[i].SourceURL = sourceURL
	}

	rssCache.Lock()
	rssCache.entries[sourceURL] = rssCacheEntry{items: items, expires: time.Now().Add(15 * time.Minute)}
	rssCache.Unlock()

	return items, nil
}

func FetchRssMulti(sources []struct{ URL, Name string }, maxItems int) []RssItem {
	type result struct {
		items []RssItem
	}
	results := make([]result, len(sources))
	var wg sync.WaitGroup
	for i, s := range sources {
		wg.Add(1)
		go func(idx int, url, name string) {
			defer wg.Done()
			items, _ := FetchRss(url, name)
			results[idx] = result{items: items}
		}(i, s.URL, s.Name)
	}
	wg.Wait()

	var all []RssItem
	for _, r := range results {
		all = append(all, r.items...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].PubDate > all[j].PubDate
	})

	if maxItems > 0 && len(all) > maxItems {
		all = all[:maxItems]
	}
	return all
}

func parseRssBody(data []byte) ([]RssItem, string, error) {
	var probe struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &probe); err != nil {
		return nil, "", err
	}

	if probe.XMLName.Local == "feed" {
		var feed atomFeed
		if err := xml.Unmarshal(data, &feed); err != nil {
			return nil, "", err
		}
		items := make([]RssItem, 0, len(feed.Entries))
		for _, e := range feed.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = l.Href
					break
				}
			}
			if link == "" {
				for _, l := range e.Links {
					link = l.Href
					break
				}
			}
			desc := e.Summary
			if desc == "" {
				desc = e.Content
			}
			pub := e.Published
			if pub == "" {
				pub = e.Updated
			}
			items = append(items, RssItem{
				Title:       strings.TrimSpace(e.Title),
				Link:        link,
				Description: stripHTML(desc),
				PubDate:     normalizeDate(pub),
			})
		}
		return items, strings.TrimSpace(feed.Title), nil
	}

	// Default: RSS 2.0
	var feed rss20Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, "", err
	}
	items := make([]RssItem, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		items = append(items, RssItem{
			Title:       strings.TrimSpace(item.Title),
			Link:        strings.TrimSpace(item.Link),
			Description: stripHTML(item.Description),
			PubDate:     normalizeDate(item.PubDate),
		})
	}
	return items, strings.TrimSpace(feed.Channel.Title), nil
}

func normalizeDate(s string) string {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return s
}

func stripHTML(s string) string {
	out := make([]byte, 0, len(s))
	inTag := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out = append(out, s[i])
			}
		}
	}
	result := string(out)
	entities := [][2]string{
		{"&amp;", "&"}, {"&lt;", "<"}, {"&gt;", ">"}, {"&quot;", `"`}, {"&#39;", "'"},
		{"&nbsp;", " "}, {"&auml;", "ä"}, {"&ouml;", "ö"}, {"&uuml;", "ü"},
		{"&Auml;", "Ä"}, {"&Ouml;", "Ö"}, {"&Uuml;", "Ü"}, {"&szlig;", "ß"},
		{"&ndash;", "–"}, {"&mdash;", "—"}, {"&hellip;", "…"},
	}
	for _, pair := range entities {
		result = strings.ReplaceAll(result, pair[0], pair[1])
	}
	return strings.Join(strings.Fields(result), " ")
}
