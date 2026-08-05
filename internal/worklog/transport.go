package worklog

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
	jar     http.CookieJar
	cookie  string
}

func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, configError("ECOLOGY_BASE 不能为空")
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, configError("ECOLOGY_BASE 不是有效 URL")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if baseTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = baseTransport.Clone()
	}
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: !cfg.VerifyTLS} // #nosec G402 -- explicit CLI setting
	client := &Client{BaseURL: base, jar: jar, cookie: strings.TrimSpace(cfg.Cookie)}
	client.HTTP = &http.Client{Transport: transport, Jar: jar, Timeout: cfg.Timeout}
	if client.cookie != "" {
		client.setCookie(client.cookie)
	}
	return client, nil
}

func (c *Client) setCookie(raw string) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= len("cookie:") && strings.EqualFold(raw[:len("cookie:")], "cookie:") {
		raw = strings.TrimSpace(raw[len("cookie:"):])
	}
	if raw == "" {
		return
	}
	if !strings.Contains(raw, "=") {
		raw = "ecology_JSessionid=" + raw
	}
	c.cookie = raw
	parsed, err := url.Parse(c.BaseURL)
	if err != nil {
		return
	}
	var cookies []*http.Cookie
	for _, part := range strings.Split(raw, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && name != "" {
			cookies = append(cookies, &http.Cookie{Name: name, Value: value, Path: "/"})
		}
	}
	c.jar.SetCookies(parsed, cookies)
}

func (c *Client) JSessionID() string {
	parsed, err := url.Parse(c.BaseURL)
	if err == nil {
		for _, cookie := range c.jar.Cookies(parsed) {
			switch strings.ToLower(cookie.Name) {
			case "ecology_jsessionid", "jsessionid":
				return cookie.Value
			}
		}
	}
	for _, part := range strings.Split(c.cookie, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && (strings.EqualFold(name, "ecology_JSessionid") || strings.EqualFold(name, "JSESSIONID")) {
			return value
		}
	}
	return ""
}

func (c *Client) URL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.BaseURL + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) DoForm(ctx context.Context, method, path string, values url.Values) (any, error) {
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, c.URL(path), body)
	if err != nil {
		return nil, err
	}
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:148.0) Gecko/20100101 Firefox/148.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	return c.do(req, path)
}

func (c *Client) DoGet(ctx context.Context, path string, query url.Values) (any, error) {
	endpoint := c.URL(path)
	if query != nil && len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:148.0) Gecko/20100101 Firefox/148.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	return c.do(req, path)
}

func (c *Client) do(req *http.Request, path string) (any, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, networkError(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, networkError(err)
	}
	if resp.StatusCode >= 400 {
		return nil, remoteError("HTTP %d %s: %s", resp.StatusCode, path, truncate(string(data), 300))
	}
	if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
		// The jar receives Set-Cookie automatically; retaining the raw header also
		// keeps compatibility with servers that use a non-standard session name.
		c.cookie = joinCookieHeader(c.cookie, cookie)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, remoteError("接口返回非 JSON: %s", truncate(string(data), 300))
	}
	return value, nil
}

func joinCookieHeader(existing, setCookie string) string {
	nameValue := strings.SplitN(setCookie, ";", 2)[0]
	name, _, ok := strings.Cut(nameValue, "=")
	if !ok {
		return existing
	}
	parts := []string{}
	for _, item := range strings.Split(existing, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		oldName, _, hasValue := strings.Cut(item, "=")
		if hasValue && !strings.EqualFold(oldName, name) {
			parts = append(parts, item)
		}
	}
	parts = append(parts, strings.TrimSpace(nameValue))
	return strings.Join(parts, "; ")
}

func formValues(values map[string]any) url.Values {
	result := url.Values{}
	for key, value := range values {
		if value == nil {
			result.Set(key, "")
			continue
		}
		result.Set(key, fmt.Sprint(value))
	}
	return result
}

func nowMillis() string { return fmt.Sprint(time.Now().UnixMilli()) }

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}
