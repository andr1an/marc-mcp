package marc

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/andr1an/marc-mcp/internal/cache"
	"golang.org/x/net/html"
)

const (
	baseURL = "https://marc.info/"

	defaultTimeout = 60 * time.Second
	minTimeout     = 10 * time.Second
	maxTimeout     = 15 * time.Minute
)

type Client struct {
	http   *http.Client
	cache  *cache.Cache
	logger *slog.Logger
}

func getTimeout() time.Duration {
	envVal := os.Getenv("MARC_TIMEOUT")
	if envVal == "" {
		return defaultTimeout
	}

	d, err := time.ParseDuration(envVal)
	if err != nil {
		return defaultTimeout
	}

	if d < minTimeout {
		return minTimeout
	}
	if d > maxTimeout {
		return maxTimeout
	}
	return d
}

func NewClient() (*Client, error) {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	c, err := cache.New(cache.Options{
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}

	return &Client{
		http:   &http.Client{Timeout: getTimeout()},
		cache:  c,
		logger: logger,
	}, nil
}

func (c *Client) Close() error {
	return c.cache.Close()
}

type MailingList struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Message struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	List    string `json:"list"`
}

type MessageContent struct {
	Message
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

func (c *Client) fetch(path string) (*html.Node, error) {
	fullURL := baseURL + path
	c.logger.Debug("fetching", "url", fullURL)

	resp, err := c.http.Get(fullURL)
	if err != nil {
		c.logger.Debug("fetch failed", "error", err)
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return html.Parse(resp.Body)
}

func (c *Client) fetchRaw(path string) (string, error) {
	fullURL := baseURL + path
	c.logger.Debug("fetching raw", "url", fullURL)

	resp, err := c.http.Get(fullURL)
	if err != nil {
		c.logger.Debug("fetch failed", "error", err)
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	return string(body), nil
}

func (c *Client) ListMailingLists() ([]MailingList, error) {
	c.logger.Debug("listing mailing lists")

	// Check cache first
	if cached, ok := c.cache.GetMailingLists(); ok {
		lists := make([]MailingList, len(cached))
		for i, cl := range cached {
			lists[i] = MailingList{Name: cl.Name, Category: cl.Category}
		}
		return lists, nil
	}

	doc, err := c.fetch("")
	if err != nil {
		return nil, err
	}

	var lists []MailingList
	var currentCategory string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// Category headers are in <dt><b><img alt="Group: "> CategoryName</b>
		if n.Type == html.ElementNode && n.Data == "dt" {
			cat := extractCategory(n)
			if cat != "" {
				currentCategory = cat
				c.logger.Debug("found category", "name", currentCategory)
			}
		}

		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if strings.HasPrefix(href, "?l=") {
				listName := extractListName(href)
				if listName != "" {
					lists = append(lists, MailingList{
						Name:     listName,
						Category: currentCategory,
					})
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	c.logger.Debug("found mailing lists", "count", len(lists))

	// Store in cache
	cacheLists := make([]cache.MailingList, len(lists))
	for i, l := range lists {
		cacheLists[i] = cache.MailingList{Name: l.Name, Category: l.Category}
	}
	c.cache.SetMailingLists(cacheLists)

	return lists, nil
}

func extractCategory(dt *html.Node) string {
	// Look for <b> inside <dt>
	for child := dt.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "b" {
			// Check if it contains an img with alt="Group: "
			hasGroupImg := false
			for c := child.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "img" {
					alt := getAttr(c, "alt")
					if strings.Contains(alt, "Group") {
						hasGroupImg = true
						break
					}
				}
			}
			if hasGroupImg {
				// Extract the text after the img
				var text string
				for c := child.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						text += c.Data
					}
				}
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

type ListMessagesOptions struct {
	List  string
	Month string // YYYYMM format
	Page  int    // Page number (1-based, default 1)
	Limit int    // Max messages to return (0 = all)
}

func (c *Client) ListMessages(list string, month string) ([]Message, error) {
	return c.ListMessagesWithOptions(ListMessagesOptions{
		List:  list,
		Month: month,
	})
}

func (c *Client) ListMessagesWithOptions(opts ListMessagesOptions) ([]Message, error) {
	// Default to current month if not specified
	if opts.Month == "" {
		opts.Month = time.Now().Format("200601")
	}

	// Default to page 1
	if opts.Page < 1 {
		opts.Page = 1
	}

	c.logger.Debug("listing messages", "list", opts.List, "month", opts.Month, "page", opts.Page, "limit", opts.Limit)

	// Check cache first (only for first page without limit)
	if opts.Page == 1 && opts.Limit == 0 {
		if cached, ok := c.cache.GetMessages(opts.List, opts.Month); ok {
			messages := make([]Message, len(cached))
			for i, cm := range cached {
				messages[i] = Message{ID: cm.ID, List: cm.List, Subject: cm.Subject, Author: cm.Author, Date: cm.Date}
			}
			return messages, nil
		}
	}

	// Build URL - r=N is page number
	path := fmt.Sprintf("?l=%s&b=%s&r=%d&w=2", url.QueryEscape(opts.List), url.QueryEscape(opts.Month), opts.Page)

	raw, err := c.fetchRaw(path)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("response length", "bytes", len(raw))

	if strings.Contains(raw, "No such list") {
		c.logger.Debug("list not found", "list", opts.List)
		return nil, fmt.Errorf("no such list: %s", opts.List)
	}

	messages := parseMessageListFromRaw(raw, opts.List, c.logger)
	c.logger.Debug("found messages", "count", len(messages))

	// Apply limit if specified
	if opts.Limit > 0 && len(messages) > opts.Limit {
		messages = messages[:opts.Limit]
	}

	// Store in cache
	cacheMessages := make([]cache.Message, len(messages))
	for i, m := range messages {
		cacheMessages[i] = cache.Message{ID: m.ID, List: m.List, Subject: m.Subject, Author: m.Author, Date: m.Date}
	}
	c.cache.SetMessages(cacheMessages)

	return messages, nil
}

func (c *Client) GetMessage(list, messageID string) (*MessageContent, error) {
	c.logger.Debug("getting message", "list", list, "messageID", messageID)

	// Check cache first
	if cached, ok := c.cache.GetMessageContent(list, messageID); ok {
		return &MessageContent{
			Message: Message{ID: cached.ID, List: cached.List, Subject: cached.Subject, Author: cached.Author, Date: cached.Date},
			Body:    cached.Body,
			Headers: cached.Headers,
		}, nil
	}

	path := fmt.Sprintf("?l=%s&m=%s&w=2", url.QueryEscape(list), url.QueryEscape(messageID))

	raw, err := c.fetchRaw(path)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("response length", "bytes", len(raw))

	msg, err := parseMessage(raw, list, messageID)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("parsed message", "subject", msg.Subject, "author", msg.Author)

	// Store in cache
	c.cache.SetMessageContent(&cache.MessageContent{
		Message: cache.Message{ID: msg.ID, List: msg.List, Subject: msg.Subject, Author: msg.Author, Date: msg.Date},
		Body:    msg.Body,
		Headers: msg.Headers,
	})

	return msg, nil
}

func (c *Client) Search(list, query, searchType string) ([]Message, error) {
	c.logger.Debug("searching", "list", list, "query", query, "type", searchType)

	// searchType: s=subject, a=author, b=body
	if searchType == "" {
		searchType = "s"
	}

	path := fmt.Sprintf("?l=%s&s=%s&q=%s&w=2",
		url.QueryEscape(list),
		url.QueryEscape(query),
		url.QueryEscape(searchType))

	doc, err := c.fetch(path)
	if err != nil {
		return nil, err
	}

	messages := parseMessageList(doc, list)
	c.logger.Debug("found messages", "count", len(messages))
	return messages, nil
}

var (
	// Match message links: href="?l=list&m=123456&w=2"
	messageLinkRegex = regexp.MustCompile(`\?l=([^&]+)&m=(\d+)`)
	// Match date at start of line: 2026-02-24
	dateRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	// Match a message line in the <pre> block
	// Format: N. YYYY-MM-DD [thread] <a href="?l=list&m=ID">Subject</a> <a href="...">list</a>  Author
	messageLineRegex = regexp.MustCompile(`^\s*\d+\.\s+(\d{4}-\d{2}-\d{2})\s+`)
)

func parseMessageListFromRaw(raw, list string, logger *slog.Logger) []Message {
	messages := make([]Message, 0)

	// Find <pre> content and process line by line
	// The messages are in format:
	//   1. 2026-02-24  [1] <a href="?l=git&m=123">Subject</a> <a href="?l=git&w=2">git</a>  Author

	// Split by lines and process each
	lines := strings.Split(raw, "\n")
	logger.Debug("processing lines", "total", len(lines))

	for _, line := range lines {
		// Skip lines that don't start with a number (message lines start with "  N. ")
		if !messageLineRegex.MatchString(line) {
			continue
		}

		// Extract date
		dateMatch := dateRegex.FindString(line)
		if dateMatch == "" {
			continue
		}

		// Extract message ID and subject from the message link
		msgMatch := messageLinkRegex.FindStringSubmatch(line)
		if len(msgMatch) != 3 {
			continue
		}
		msgID := msgMatch[2]

		// Extract subject from the link text: <a href="...">Subject</a>
		// Find the content between > and </a> after the message link
		linkStart := strings.Index(line, "?l="+list+"&m="+msgID)
		if linkStart == -1 {
			continue
		}

		// Find the closing > of the <a> tag
		subjectStart := strings.Index(line[linkStart:], ">")
		if subjectStart == -1 {
			continue
		}
		subjectStart += linkStart + 1

		// Find the </a>
		subjectEnd := strings.Index(line[subjectStart:], "</a>")
		if subjectEnd == -1 {
			continue
		}

		subject := strings.TrimSpace(line[subjectStart : subjectStart+subjectEnd])

		// Extract author - it's after the last </a> and the list name
		// Pattern: </a> <a href="?l=git&w=2">git</a>       Author Name
		lastAnchorEnd := strings.LastIndex(line, "</a>")
		author := ""
		if lastAnchorEnd != -1 && lastAnchorEnd+4 < len(line) {
			author = strings.TrimSpace(line[lastAnchorEnd+4:])
		}

		msg := Message{
			ID:      msgID,
			Subject: subject,
			Author:  author,
			Date:    dateMatch,
			List:    list,
		}

		logger.Debug("parsed message", "id", msgID, "date", dateMatch, "subject", subject[:min(30, len(subject))], "author", author)
		messages = append(messages, msg)
	}

	return messages
}

// Keep old function for backward compatibility with Search
func parseMessageList(doc *html.Node, list string) []Message {
	messages := make([]Message, 0)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if matches := messageLinkRegex.FindStringSubmatch(href); len(matches) == 3 {
				subject := strings.TrimSpace(extractText(n))
				if subject != "" {
					msg := Message{
						ID:      matches[2],
						Subject: subject,
						List:    list,
					}
					msg.Date, msg.Author = extractMessageMetaSimple(n)
					messages = append(messages, msg)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return messages
}

func extractMessageMetaSimple(n *html.Node) (date, author string) {
	// Walk up to find parent context
	var lineText string
	if n.Parent != nil {
		lineText = extractText(n.Parent)
	}

	if dateMatch := dateRegex.FindString(lineText); dateMatch != "" {
		date = dateMatch
	}

	if idx := strings.LastIndex(lineText, "]"); idx >= 0 {
		author = strings.TrimSpace(lineText[idx+1:])
		author = strings.TrimSuffix(author, "Next")
		author = strings.TrimSuffix(author, "Last")
		author = strings.TrimSpace(author)
	}

	return
}

func parseMessage(raw, list, messageID string) (*MessageContent, error) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil, err
	}

	msg := &MessageContent{
		Message: Message{
			ID:   messageID,
			List: list,
		},
		Headers: make(map[string]string),
	}

	var inPre bool
	var preContent strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "pre" {
			inPre = true
		}

		if inPre && n.Type == html.TextNode {
			preContent.WriteString(n.Data)
		}

		if n.Type == html.ElementNode && n.Data == "pre" {
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			inPre = false
			return
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	content := preContent.String()
	lines := strings.Split(content, "\n")

	// Parse headers and body
	inHeaders := true
	var bodyLines []string

	for _, line := range lines {
		if inHeaders {
			if line == "" {
				inHeaders = false
				continue
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				value := strings.TrimSpace(line[idx+1:])
				msg.Headers[key] = value

				switch strings.ToLower(key) {
				case "subject":
					msg.Subject = value
				case "from":
					msg.Author = value
				case "date":
					msg.Date = value
				}
			}
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	msg.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return msg, nil
}

func extractText(n *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return strings.TrimSpace(text.String())
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func extractListName(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return u.Query().Get("l")
}

func extractMessageID(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return u.Query().Get("m")
}
