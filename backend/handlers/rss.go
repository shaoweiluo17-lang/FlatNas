package handlers

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"golang.org/x/net/html/charset"
)

// RssPayload defines the input structure
type RssPayload struct {
	Url string `json:"url"`
}

// Unified Item structure for frontend
type UnifiedRssItem struct {
	Title          string `json:"title"`
	Link           string `json:"link"`
	PubDate        string `json:"pubDate"`
	ContentSnippet string `json:"contentSnippet"`
}

// Cache structures
type CachedRssItem struct {
	Items     []UnifiedRssItem
	ExpiresAt time.Time
}

var (
	rssCache = make(map[string]CachedRssItem)
	rssCacheMutex sync.RWMutex
	RssCacheTTL = 6 * time.Hour
)

// RSS 2.0 Structures
type Rss2Feed struct {
	Channel Rss2Channel `xml:"channel"`
}

type Rss2Channel struct {
	Items []Rss2Item `xml:"item"`
}

type Rss2Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Atom Structures
type AtomFeed struct {
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	Title   string    `xml:"title"`
	Link    AtomLink  `xml:"link"`
	Content string    `xml:"content"`
	Summary string    `xml:"summary"`
	Updated string    `xml:"updated"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

func BindRssHandlers(server *socketio.Server) {
	server.OnEvent("/", "rss:fetch", func(s socketio.Conn, msg interface{}) {
		log.Println("Received rss:fetch event")
		var urlStr string
		if m, ok := msg.(map[string]interface{}); ok {
			if u, ok := m["url"].(string); ok {
				urlStr = u
			}
		}

		if urlStr == "" {
			s.Emit("rss:error", map[string]interface{}{"error": "url is required"})
			return
		}

		// Check cache
		rssCacheMutex.RLock()
		cached, exists := rssCache[urlStr]
		rssCacheMutex.RUnlock()

		if exists && time.Now().Before(cached.ExpiresAt) {
			s.Emit("rss:data", map[string]interface{}{
				"url": urlStr,
				"data": map[string]interface{}{
					"items": cached.Items,
				},
			})
			return
		}

		items, err := fetchRssFeed(urlStr)
		if err != nil {
			s.Emit("rss:error", map[string]interface{}{"url": urlStr, "error": err.Error()})
			return
		}

		// Update cache
		rssCacheMutex.Lock()
		rssCache[urlStr] = CachedRssItem{
			Items:     items,
			ExpiresAt: time.Now().Add(RssCacheTTL),
		}
		rssCacheMutex.Unlock()

		s.Emit("rss:data", map[string]interface{}{
			"url": urlStr,
			"data": map[string]interface{}{
				"items": items,
			},
		})
	})
}

func fetchRssFeed(feedUrl string) ([]UnifiedRssItem, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", feedUrl, nil)
	if err != nil {
		return nil, err
	}
	
	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try RSS 2.0 first
	var rss2 Rss2Feed
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&rss2); err == nil && len(rss2.Channel.Items) > 0 {
		items := make([]UnifiedRssItem, 0, len(rss2.Channel.Items))
		for _, item := range rss2.Channel.Items {
			desc := cleanDescription(item.Description)
			items = append(items, UnifiedRssItem{
				Title:          item.Title,
				Link:           item.Link,
				PubDate:        item.PubDate,
				ContentSnippet: desc,
			})
		}
		return items, nil
	}

	// Try Atom
	var atom AtomFeed
	decoder = xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&atom); err == nil && len(atom.Entries) > 0 {
		items := make([]UnifiedRssItem, 0, len(atom.Entries))
		for _, entry := range atom.Entries {
			desc := cleanDescription(entry.Summary)
			if desc == "" {
				desc = cleanDescription(entry.Content)
			}
			items = append(items, UnifiedRssItem{
				Title:          entry.Title,
				Link:           entry.Link.Href,
				PubDate:        entry.Updated,
				ContentSnippet: desc,
			})
		}
		return items, nil
	}

	return nil, fmt.Errorf("failed to parse feed")
}

func cleanDescription(html string) string {
	// Simple strip tags
	// In a real app we might want a proper HTML sanitizer, but here we just strip generic tags
	// Or just return truncated text
	
	// Remove <![CDATA[ ... ]]> wrapper
	if strings.HasPrefix(html, "<![CDATA[") && strings.HasSuffix(html, "]]>") {
		html = html[9 : len(html)-3]
	}

	// Very basic tag stripping (naive)
	// Replace <br> with space
	html = strings.ReplaceAll(html, "<br>", " ")
	html = strings.ReplaceAll(html, "<br/>", " ")
	
	// Remove other tags (naive regex)
	// Note: regex in Go for HTML is not perfect but sufficient for snippets
	// Ideally use a library like bluemonday, but we avoid new deps
	
	// Truncate to 100 chars
	runes := []rune(html)
	if len(runes) > 100 {
		return string(runes[:100]) + "..."
	}
	return html
}
