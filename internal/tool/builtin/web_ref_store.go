package builtin

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

const maxOpenLinks = 100

type webPageRef struct {
	RefID   string                   `json:"ref_id"`
	URL     string                   `json:"url"`
	Status  string                   `json:"status"`
	Content string                   `json:"content"`
	Links   []map[string]interface{} `json:"links"`
}

type webSearchRef struct {
	RefID string `json:"ref_id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

var (
	anchorTagRe  = regexp.MustCompile(`(?is)<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	htmlStripRe2 = regexp.MustCompile(`(?is)<[^>]+>`)
)

type webRefState struct {
	mu         sync.RWMutex
	nextFetch  int64
	nextSearch int64
	pages      map[string]*webPageRef
	searchRefs map[string]*webSearchRef
}

var globalWebRefState = &webRefState{
	nextFetch:  -1,
	nextSearch: -1,
	pages:      make(map[string]*webPageRef),
	searchRefs: make(map[string]*webSearchRef),
}

func storeWebPage(urlValue, status, content string, links []map[string]interface{}) string {
	id := atomic.AddInt64(&globalWebRefState.nextFetch, 1)
	refID := "turn0fetch" + formatInt64(id)

	globalWebRefState.mu.Lock()
	globalWebRefState.pages[refID] = &webPageRef{
		RefID:   refID,
		URL:     urlValue,
		Status:  status,
		Content: content,
		Links:   links,
	}
	globalWebRefState.mu.Unlock()
	return refID
}

func getWebPage(refID string) (*webPageRef, bool) {
	globalWebRefState.mu.RLock()
	defer globalWebRefState.mu.RUnlock()
	ref, ok := globalWebRefState.pages[refID]
	return ref, ok
}

func storeWebSearchRef(urlValue, title string) string {
	id := atomic.AddInt64(&globalWebRefState.nextSearch, 1)
	refID := "turn0search" + formatInt64(id)

	globalWebRefState.mu.Lock()
	globalWebRefState.searchRefs[refID] = &webSearchRef{
		RefID: refID,
		URL:   strings.TrimSpace(urlValue),
		Title: strings.TrimSpace(title),
	}
	globalWebRefState.mu.Unlock()
	return refID
}

func getWebSearchRef(refID string) (*webSearchRef, bool) {
	globalWebRefState.mu.RLock()
	defer globalWebRefState.mu.RUnlock()
	ref, ok := globalWebRefState.searchRefs[refID]
	return ref, ok
}

func parseOpenLinks(baseURL, doc string) []map[string]interface{} {
	matches := anchorTagRe.FindAllStringSubmatch(doc, maxOpenLinks)
	if len(matches) == 0 {
		return nil
	}

	base, _ := url.Parse(baseURL)
	out := make([]map[string]interface{}, 0, len(matches))

	for idx, m := range matches {
		if len(m) < 3 {
			continue
		}
		href := strings.TrimSpace(html.UnescapeString(m[1]))
		if href == "" {
			continue
		}
		if base != nil {
			if u, err := base.Parse(href); err == nil {
				href = u.String()
			}
		}
		text := strings.TrimSpace(html.UnescapeString(htmlStripRe2.ReplaceAllString(m[2], "")))
		out = append(out, map[string]interface{}{
			"id":   idx + 1,
			"text": text,
			"url":  href,
		})
	}

	return out
}

func formatInt64(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if negative {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
