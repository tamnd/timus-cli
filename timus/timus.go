// Package timus is the library behind the timus command line:
// the HTTP client, request shaping, and the typed data models for Timus
// Online Judge problems fetched by scraping acm.timus.ru.
package timus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to Timus Online Judge.
const DefaultUserAgent = "timus-cli/0.1 (+https://github.com/tamnd/timus-cli)"

// problemRe extracts num, title, solved count, and difficulty from a row.
var problemRe = regexp.MustCompile(`<TD>(\d{4,})</TD><TD CLASS="name"><A HREF="problem\.aspx\?space=1&amp;num=\d+">([^<]+)</A></TD><TD[^>]*>.*?<TD><A HREF="rating[^"]+">(\d+)</A></TD><TD>(\d+)</TD>`)

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://acm.timus.ru",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to Timus Online Judge over HTTP.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		b, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return b, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get: %w", lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Client) fetchPage(ctx context.Context, page int) ([]Problem, bool, error) {
	url := fmt.Sprintf("%s/problemset.aspx?space=1&page=%d", c.cfg.BaseURL, page)
	raw, err := c.get(ctx, url)
	if err != nil {
		return nil, false, err
	}
	html := string(raw)
	matches := problemRe.FindAllStringSubmatch(html, -1)
	var out []Problem
	for _, m := range matches {
		num, _ := strconv.Atoi(m[1])
		solved, _ := strconv.Atoi(m[3])
		diff, _ := strconv.Atoi(m[4])
		out = append(out, Problem{
			Num:        num,
			Title:      m[2],
			Solved:     solved,
			Difficulty: diff,
			URL:        fmt.Sprintf("https://acm.timus.ru/problem.aspx?space=1&num=%d", num),
		})
	}
	hasMore := strings.Contains(html, fmt.Sprintf("page=%d", page+1))
	return out, hasMore, nil
}

// List fetches all problems, up to limit (0 = all).
func (c *Client) List(ctx context.Context, limit int) ([]Problem, error) {
	var all []Problem
	for page := 1; ; page++ {
		probs, hasMore, err := c.fetchPage(ctx, page)
		if err != nil {
			return nil, err
		}
		for _, p := range probs {
			all = append(all, p)
			if limit > 0 && len(all) >= limit {
				goto done
			}
		}
		if !hasMore || len(probs) == 0 {
			break
		}
		if c.cfg.Rate > 0 {
			time.Sleep(c.cfg.Rate)
		}
	}
done:
	for i := range all {
		all[i].Rank = i + 1
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// Search fetches all problems and filters by title substring (case-insensitive).
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Problem, error) {
	all, err := c.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var out []Problem
	rank := 0
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.Title), q) {
			rank++
			p.Rank = rank
			out = append(out, p)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
