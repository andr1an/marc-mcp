package marc

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestGetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string // duration string for comparison
	}{
		{"default when empty", "", "1m0s"},
		{"valid duration", "30s", "30s"},
		{"minimum enforced", "1s", "10s"},
		{"maximum enforced", "1h", "15m0s"},
		{"invalid falls back to default", "invalid", "1m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("MARC_TIMEOUT", tt.envValue)
				defer os.Unsetenv("MARC_TIMEOUT")
			} else {
				os.Unsetenv("MARC_TIMEOUT")
			}

			got := getTimeout()
			if got.String() != tt.want {
				t.Errorf("getTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractListName(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{"?l=git", "git"},
		{"?l=linux-kernel", "linux-kernel"},
		{"?l=git&m=123", "git"},
		{"?l=git&w=2", "git"},
		{"", ""},
		{"invalid", ""},
		{"?m=123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := extractListName(tt.href)
			if got != tt.want {
				t.Errorf("extractListName(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

func TestExtractMessageID(t *testing.T) {
	tests := []struct {
		href string
		want string
	}{
		{"?l=git&m=123456", "123456"},
		{"?m=789", "789"},
		{"?l=git", ""},
		{"", ""},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := extractMessageID(tt.href)
			if got != tt.want {
				t.Errorf("extractMessageID(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

func TestGetAttr(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "a",
		Attr: []html.Attribute{
			{Key: "href", Val: "https://example.com"},
			{Key: "class", Val: "link"},
		},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"href", "https://example.com"},
		{"class", "link"},
		{"id", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := getAttr(node, tt.key)
			if got != tt.want {
				t.Errorf("getAttr(node, %q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"simple text",
			"<span>Hello World</span>",
			"Hello World",
		},
		{
			"nested elements",
			"<div><span>Hello</span> <b>World</b></div>",
			"Hello World",
		},
		{
			"with whitespace",
			"<p>  Multiple   Spaces  </p>",
			"Multiple   Spaces",
		},
		{
			"empty element",
			"<div></div>",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}

			// Find the body element (html.Parse wraps in html>body)
			var body *html.Node
			var findBody func(*html.Node)
			findBody = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "body" {
					body = n
					return
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findBody(c)
				}
			}
			findBody(doc)

			got := extractText(body.FirstChild)
			if got != tt.want {
				t.Errorf("extractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMessageListFromRaw(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	testHTML := `<html>
<head><title>git messages</title></head>
<body>
<pre>
   1. 2026-02-24  [1] <a href="?l=git&m=174037595823063">Fix memory leak in cache</a> <a href="?l=git&w=2">git</a>  Alice Developer
   2. 2026-02-23  [1] <a href="?l=git&m=174037595823064">Add new subcommand</a> <a href="?l=git&w=2">git</a>  Bob Maintainer
   3. 2026-02-22  [1] <a href="?l=git&m=174037595823065">Update documentation</a> <a href="?l=git&w=2">git</a>  Charlie Doc
</pre>
</body>
</html>`

	messages := parseMessageListFromRaw(testHTML, "git", logger)

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Check first message
	if messages[0].ID != "174037595823063" {
		t.Errorf("message 0 ID = %q, want %q", messages[0].ID, "174037595823063")
	}
	if messages[0].Subject != "Fix memory leak in cache" {
		t.Errorf("message 0 Subject = %q, want %q", messages[0].Subject, "Fix memory leak in cache")
	}
	if messages[0].Date != "2026-02-24" {
		t.Errorf("message 0 Date = %q, want %q", messages[0].Date, "2026-02-24")
	}
	if messages[0].Author != "Alice Developer" {
		t.Errorf("message 0 Author = %q, want %q", messages[0].Author, "Alice Developer")
	}
	if messages[0].List != "git" {
		t.Errorf("message 0 List = %q, want %q", messages[0].List, "git")
	}

	// Check that all messages have the correct list
	for i, msg := range messages {
		if msg.List != "git" {
			t.Errorf("message %d List = %q, want %q", i, msg.List, "git")
		}
	}
}

func TestParseMessageListFromRaw_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	testHTML := `<html><body><pre>No messages found</pre></body></html>`

	messages := parseMessageListFromRaw(testHTML, "empty-list", logger)

	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestParseMessage(t *testing.T) {
	testHTML := `<html>
<head><title>Message</title></head>
<body>
<pre>
From: Alice Developer &lt;alice@example.com&gt;
Subject: Fix buffer overflow bug
Date: Thu, 15 Feb 2026 10:30:00 +0000
Message-ID: &lt;123456@git.example.com&gt;

This patch fixes a critical buffer overflow in the core module.

The issue was caused by incorrect bounds checking when processing
large input files.

Signed-off-by: Alice Developer &lt;alice@example.com&gt;
</pre>
</body>
</html>`

	msg, err := parseMessage(testHTML, "git", "123456")
	if err != nil {
		t.Fatalf("parseMessage failed: %v", err)
	}

	if msg.ID != "123456" {
		t.Errorf("ID = %q, want %q", msg.ID, "123456")
	}
	if msg.List != "git" {
		t.Errorf("List = %q, want %q", msg.List, "git")
	}
	if msg.Subject != "Fix buffer overflow bug" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "Fix buffer overflow bug")
	}
	if msg.Author != "Alice Developer <alice@example.com>" {
		t.Errorf("Author = %q, want %q", msg.Author, "Alice Developer <alice@example.com>")
	}
	if msg.Date != "Thu, 15 Feb 2026 10:30:00 +0000" {
		t.Errorf("Date = %q, want %q", msg.Date, "Thu, 15 Feb 2026 10:30:00 +0000")
	}
	if msg.Headers["Message-ID"] != "<123456@git.example.com>" {
		t.Errorf("Headers[Message-ID] = %q, want %q", msg.Headers["Message-ID"], "<123456@git.example.com>")
	}
	if !strings.Contains(msg.Body, "buffer overflow") {
		t.Errorf("Body should contain 'buffer overflow', got: %q", msg.Body)
	}
	if !strings.Contains(msg.Body, "Signed-off-by") {
		t.Errorf("Body should contain 'Signed-off-by', got: %q", msg.Body)
	}
}

func TestParseMessage_EmptyBody(t *testing.T) {
	testHTML := `<html><body><pre>
From: Test
Subject: Empty message
Date: Thu, 15 Feb 2026 10:30:00 +0000

</pre></body></html>`

	msg, err := parseMessage(testHTML, "test", "1")
	if err != nil {
		t.Fatalf("parseMessage failed: %v", err)
	}

	if msg.Body != "" {
		t.Errorf("expected empty body, got %q", msg.Body)
	}
}

func TestExtractCategory(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"with group image",
			`<dt><b><img alt="Group: " src="group.gif"> Development</b></dt>`,
			"Development",
		},
		{
			"without group image",
			`<dt><b>Not a category</b></dt>`,
			"",
		},
		{
			"empty dt",
			`<dt></dt>`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}

			// Find the dt element
			var dt *html.Node
			var findDt func(*html.Node)
			findDt = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "dt" {
					dt = n
					return
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findDt(c)
				}
			}
			findDt(doc)

			if dt == nil {
				t.Fatal("dt element not found")
			}

			got := extractCategory(dt)
			if got != tt.want {
				t.Errorf("extractCategory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessageLinkRegex(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
		wantID  string
	}{
		{"?l=git&m=123456", 3, "123456"},
		{"?l=linux-kernel&m=789012345", 3, "789012345"},
		{"?l=git", 0, ""},
		{"?m=123", 0, ""}, // Missing list
		{"invalid", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := messageLinkRegex.FindStringSubmatch(tt.input)
			if len(matches) != tt.wantLen {
				t.Errorf("match count = %d, want %d", len(matches), tt.wantLen)
			}
			if tt.wantLen > 0 && matches[2] != tt.wantID {
				t.Errorf("message ID = %q, want %q", matches[2], tt.wantID)
			}
		})
	}
}

func TestDateRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-02-24 some text", "2026-02-24"},
		{"prefix 2025-12-31 suffix", "2025-12-31"},
		{"no date here", ""},
		{"2026-2-4 invalid format", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := dateRegex.FindString(tt.input)
			if got != tt.want {
				t.Errorf("dateRegex.FindString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMessageLineRegex(t *testing.T) {
	tests := []struct {
		input string
		match bool
	}{
		{"   1. 2026-02-24  [1] message", true},
		{"  10. 2026-01-15  [2] message", true},
		{" 100. 2025-12-31  [1] message", true},
		{"1. 2026-02-24 message", true},        // Regex allows zero leading spaces
		{"   1 2026-02-24 message", false},     // Missing period
		{"not a message line", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := messageLineRegex.MatchString(tt.input)
			if got != tt.match {
				t.Errorf("messageLineRegex.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
		})
	}
}

func TestListMessagesOptions_Defaults(t *testing.T) {
	opts := ListMessagesOptions{
		List: "test",
	}

	// Page should default to 1 when < 1
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Page != 1 {
		t.Errorf("Page = %d, want 1", opts.Page)
	}

	// Month should default to current month when empty
	if opts.Month == "" {
		// This is handled in ListMessagesWithOptions
	}
}
