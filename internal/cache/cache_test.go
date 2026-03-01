package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	c, err := New(Options{
		DBPath: dbPath,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	t.Cleanup(func() {
		c.Close()
	})

	return c
}

func TestNew(t *testing.T) {
	t.Run("creates cache with custom path", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "subdir", "cache.db")

		c, err := New(Options{DBPath: dbPath})
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}
		defer c.Close()

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}
	})

	t.Run("creates cache with default path", func(t *testing.T) {
		// Skip if we can't get user cache dir
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			t.Skip("cannot get user cache dir")
		}

		// Skip if cache dir is not writable (e.g., in Nix sandbox)
		testDir := filepath.Join(cacheDir, "marc-mcp-test-write-check")
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Skipf("cache dir not writable: %v", err)
		}
		os.RemoveAll(testDir)

		// Clean up after test
		testDBPath := filepath.Join(cacheDir, "marc-mcp", "cache.db")
		defer os.RemoveAll(filepath.Dir(testDBPath))

		c, err := New(Options{})
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}
		defer c.Close()
	})
}

func TestMailingLists(t *testing.T) {
	c := newTestCache(t)

	t.Run("returns false when empty", func(t *testing.T) {
		lists, ok := c.GetMailingLists()
		if ok {
			t.Error("expected cache miss on empty cache")
		}
		if lists != nil {
			t.Errorf("expected nil lists, got %v", lists)
		}
	})

	t.Run("stores and retrieves lists", func(t *testing.T) {
		testLists := []MailingList{
			{Name: "git", Category: "Development"},
			{Name: "linux-kernel", Category: "Linux"},
			{Name: "openssh", Category: "Security"},
		}

		if err := c.SetMailingLists(testLists); err != nil {
			t.Fatalf("failed to set mailing lists: %v", err)
		}

		lists, ok := c.GetMailingLists()
		if !ok {
			t.Fatal("expected cache hit")
		}

		if len(lists) != len(testLists) {
			t.Errorf("expected %d lists, got %d", len(testLists), len(lists))
		}

		// Results are ordered by category, name
		expected := []MailingList{
			{Name: "git", Category: "Development"},
			{Name: "linux-kernel", Category: "Linux"},
			{Name: "openssh", Category: "Security"},
		}

		for i, l := range lists {
			if l.Name != expected[i].Name || l.Category != expected[i].Category {
				t.Errorf("list %d: expected %+v, got %+v", i, expected[i], l)
			}
		}
	})

	t.Run("updates existing lists", func(t *testing.T) {
		// Use fresh cache for this test
		c2 := newTestCache(t)

		initial := []MailingList{
			{Name: "git", Category: "Development"},
		}
		if err := c2.SetMailingLists(initial); err != nil {
			t.Fatalf("failed to set initial lists: %v", err)
		}

		updated := []MailingList{
			{Name: "git", Category: "VCS"},
		}
		if err := c2.SetMailingLists(updated); err != nil {
			t.Fatalf("failed to set updated lists: %v", err)
		}

		lists, ok := c2.GetMailingLists()
		if !ok {
			t.Fatal("expected cache hit")
		}

		if len(lists) != 1 {
			t.Errorf("expected 1 list, got %d", len(lists))
		}

		if lists[0].Category != "VCS" {
			t.Errorf("expected category 'VCS', got '%s'", lists[0].Category)
		}
	})
}

func TestMessages(t *testing.T) {
	c := newTestCache(t)

	t.Run("returns false when empty", func(t *testing.T) {
		messages, ok := c.GetMessages("git", "202602")
		if ok {
			t.Error("expected cache miss on empty cache")
		}
		if messages != nil {
			t.Errorf("expected nil messages, got %v", messages)
		}
	})

	t.Run("stores and retrieves messages", func(t *testing.T) {
		testMessages := []Message{
			{ID: "123", List: "git", Subject: "Test subject", Author: "Author 1", Date: "2026-02-15"},
			{ID: "124", List: "git", Subject: "Another subject", Author: "Author 2", Date: "2026-02-16"},
		}

		if err := c.SetMessages(testMessages); err != nil {
			t.Fatalf("failed to set messages: %v", err)
		}

		messages, ok := c.GetMessages("git", "202602")
		if !ok {
			t.Fatal("expected cache hit")
		}

		if len(messages) != len(testMessages) {
			t.Errorf("expected %d messages, got %d", len(testMessages), len(messages))
		}
	})

	t.Run("filters by list", func(t *testing.T) {
		testMessages := []Message{
			{ID: "200", List: "git", Subject: "Git message", Author: "Author", Date: "2026-02-15"},
			{ID: "201", List: "linux-kernel", Subject: "Linux message", Author: "Author", Date: "2026-02-15"},
		}

		if err := c.SetMessages(testMessages); err != nil {
			t.Fatalf("failed to set messages: %v", err)
		}

		gitMessages, ok := c.GetMessages("git", "202602")
		if !ok {
			t.Fatal("expected cache hit for git")
		}

		for _, m := range gitMessages {
			if m.List != "git" {
				t.Errorf("expected list 'git', got '%s'", m.List)
			}
		}
	})

	t.Run("filters by month", func(t *testing.T) {
		testMessages := []Message{
			{ID: "300", List: "test", Subject: "Feb message", Author: "Author", Date: "2026-02-15"},
			{ID: "301", List: "test", Subject: "Jan message", Author: "Author", Date: "2026-01-15"},
		}

		if err := c.SetMessages(testMessages); err != nil {
			t.Fatalf("failed to set messages: %v", err)
		}

		febMessages, ok := c.GetMessages("test", "202602")
		if !ok {
			t.Fatal("expected cache hit for February")
		}

		if len(febMessages) != 1 {
			t.Errorf("expected 1 message for February, got %d", len(febMessages))
		}

		if febMessages[0].ID != "300" {
			t.Errorf("expected message ID '300', got '%s'", febMessages[0].ID)
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		if err := c.SetMessages(nil); err != nil {
			t.Errorf("SetMessages(nil) should not error: %v", err)
		}
		if err := c.SetMessages([]Message{}); err != nil {
			t.Errorf("SetMessages([]) should not error: %v", err)
		}
	})
}

func TestMessageContent(t *testing.T) {
	c := newTestCache(t)

	t.Run("returns false when not found", func(t *testing.T) {
		content, ok := c.GetMessageContent("git", "nonexistent")
		if ok {
			t.Error("expected cache miss")
		}
		if content != nil {
			t.Errorf("expected nil content, got %+v", content)
		}
	})

	t.Run("stores and retrieves message content", func(t *testing.T) {
		testContent := &MessageContent{
			Message: Message{
				ID:      "12345",
				List:    "git",
				Subject: "Test commit message",
				Author:  "Test Author <test@example.com>",
				Date:    "Thu, 15 Feb 2026 10:00:00 +0000",
			},
			Body: "This is the message body.\n\nWith multiple paragraphs.",
			Headers: map[string]string{
				"Subject":    "Test commit message",
				"From":       "Test Author <test@example.com>",
				"Date":       "Thu, 15 Feb 2026 10:00:00 +0000",
				"Message-ID": "<12345@example.com>",
			},
		}

		if err := c.SetMessageContent(testContent); err != nil {
			t.Fatalf("failed to set message content: %v", err)
		}

		content, ok := c.GetMessageContent("git", "12345")
		if !ok {
			t.Fatal("expected cache hit")
		}

		if content.ID != testContent.ID {
			t.Errorf("expected ID '%s', got '%s'", testContent.ID, content.ID)
		}
		if content.Subject != testContent.Subject {
			t.Errorf("expected Subject '%s', got '%s'", testContent.Subject, content.Subject)
		}
		if content.Body != testContent.Body {
			t.Errorf("expected Body '%s', got '%s'", testContent.Body, content.Body)
		}
		if content.Headers["Message-ID"] != testContent.Headers["Message-ID"] {
			t.Errorf("expected Message-ID header '%s', got '%s'",
				testContent.Headers["Message-ID"], content.Headers["Message-ID"])
		}
	})

	t.Run("updates existing content", func(t *testing.T) {
		initial := &MessageContent{
			Message: Message{ID: "999", List: "test", Subject: "Initial"},
			Body:    "Initial body",
			Headers: map[string]string{},
		}
		if err := c.SetMessageContent(initial); err != nil {
			t.Fatalf("failed to set initial content: %v", err)
		}

		updated := &MessageContent{
			Message: Message{ID: "999", List: "test", Subject: "Updated"},
			Body:    "Updated body",
			Headers: map[string]string{"X-Custom": "value"},
		}
		if err := c.SetMessageContent(updated); err != nil {
			t.Fatalf("failed to set updated content: %v", err)
		}

		content, ok := c.GetMessageContent("test", "999")
		if !ok {
			t.Fatal("expected cache hit")
		}

		if content.Subject != "Updated" {
			t.Errorf("expected subject 'Updated', got '%s'", content.Subject)
		}
		if content.Body != "Updated body" {
			t.Errorf("expected body 'Updated body', got '%s'", content.Body)
		}
	})
}

func TestSearchMessages(t *testing.T) {
	c := newTestCache(t)

	// Populate with test data
	testContents := []*MessageContent{
		{
			Message: Message{ID: "1", List: "git", Subject: "Fix buffer overflow", Author: "Alice"},
			Body:    "This patch fixes a critical buffer overflow in the core module.",
		},
		{
			Message: Message{ID: "2", List: "git", Subject: "Add new feature", Author: "Bob"},
			Body:    "This adds a new feature for better performance.",
		},
		{
			Message: Message{ID: "3", List: "linux-kernel", Subject: "Memory leak fix", Author: "Charlie"},
			Body:    "Fixed memory leak in driver subsystem.",
		},
	}

	for _, content := range testContents {
		if err := c.SetMessageContent(content); err != nil {
			t.Fatalf("failed to set content: %v", err)
		}
	}

	t.Run("searches across all lists", func(t *testing.T) {
		results, err := c.SearchMessages("fix", "")
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if len(results) < 2 {
			t.Errorf("expected at least 2 results for 'fix', got %d", len(results))
		}
	})

	t.Run("filters by list", func(t *testing.T) {
		results, err := c.SearchMessages("fix", "git")
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		for _, r := range results {
			if r.List != "git" {
				t.Errorf("expected list 'git', got '%s'", r.List)
			}
		}
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		results, err := c.SearchMessages("nonexistentterm123", "")
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cleanup.db")

	// Create cache with 2 second TTL (Unix timestamps have second precision)
	c, err := New(Options{
		DBPath: dbPath,
		TTL:    2 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	// Add test data
	if err := c.SetMailingLists([]MailingList{{Name: "test", Category: "Test"}}); err != nil {
		t.Fatalf("failed to set lists: %v", err)
	}
	if err := c.SetMessages([]Message{{ID: "1", List: "test", Subject: "Test", Author: "A", Date: "2026-02-15"}}); err != nil {
		t.Fatalf("failed to set messages: %v", err)
	}
	if err := c.SetMessageContent(&MessageContent{Message: Message{ID: "1", List: "test"}, Body: "test", Headers: map[string]string{}}); err != nil {
		t.Fatalf("failed to set content: %v", err)
	}

	// Wait for TTL to expire (Unix timestamps have second precision, so we need 3+ seconds)
	time.Sleep(3 * time.Second)

	// Run cleanup
	if err := c.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Verify data is gone
	if lists, ok := c.GetMailingLists(); ok {
		t.Errorf("expected cache miss after cleanup, got %d lists", len(lists))
	}
	if messages, ok := c.GetMessages("test", ""); ok {
		t.Errorf("expected cache miss after cleanup, got %d messages", len(messages))
	}
	if content, ok := c.GetMessageContent("test", "1"); ok {
		t.Errorf("expected cache miss after cleanup, got %+v", content)
	}
}

func TestTTLExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ttl.db")

	// Create cache with 2 second TTL (Unix timestamps have second precision)
	c, err := New(Options{
		DBPath: dbPath,
		TTL:    2 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	// Add data
	if err := c.SetMailingLists([]MailingList{{Name: "test", Category: "Test"}}); err != nil {
		t.Fatalf("failed to set lists: %v", err)
	}

	// Should get cache hit immediately
	if _, ok := c.GetMailingLists(); !ok {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for TTL to expire (Unix timestamps have second precision, so we need 3+ seconds)
	time.Sleep(3 * time.Second)

	// Should get cache miss after TTL
	if _, ok := c.GetMailingLists(); ok {
		t.Error("expected cache miss after TTL expiration")
	}
}
