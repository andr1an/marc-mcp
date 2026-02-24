package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS mailing_lists (
	name TEXT PRIMARY KEY,
	category TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	list TEXT NOT NULL,
	subject TEXT NOT NULL,
	author TEXT NOT NULL,
	date TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS message_content (
	id TEXT PRIMARY KEY,
	list TEXT NOT NULL,
	subject TEXT NOT NULL,
	author TEXT NOT NULL,
	date TEXT NOT NULL,
	body TEXT NOT NULL,
	headers TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
	id,
	list,
	subject,
	author,
	body,
	content='message_content',
	content_rowid='rowid'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON message_content BEGIN
	INSERT INTO messages_fts(rowid, id, list, subject, author, body)
	VALUES (NEW.rowid, NEW.id, NEW.list, NEW.subject, NEW.author, NEW.body);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON message_content BEGIN
	INSERT INTO messages_fts(messages_fts, rowid, id, list, subject, author, body)
	VALUES ('delete', OLD.rowid, OLD.id, OLD.list, OLD.subject, OLD.author, OLD.body);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON message_content BEGIN
	INSERT INTO messages_fts(messages_fts, rowid, id, list, subject, author, body)
	VALUES ('delete', OLD.rowid, OLD.id, OLD.list, OLD.subject, OLD.author, OLD.body);
	INSERT INTO messages_fts(rowid, id, list, subject, author, body)
	VALUES (NEW.rowid, NEW.id, NEW.list, NEW.subject, NEW.author, NEW.body);
END;

-- Future: summaries table
CREATE TABLE IF NOT EXISTS summaries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id TEXT NOT NULL,
	summary_type TEXT NOT NULL,
	content TEXT NOT NULL,
	model TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	FOREIGN KEY (message_id) REFERENCES message_content(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_list ON messages(list);
CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date);
CREATE INDEX IF NOT EXISTS idx_message_content_list ON message_content(list);
CREATE INDEX IF NOT EXISTS idx_summaries_message ON summaries(message_id);
`

type Cache struct {
	db     *sql.DB
	logger *slog.Logger
	ttl    time.Duration
}

type Options struct {
	DBPath string
	TTL    time.Duration
	Logger *slog.Logger
}

func New(opts Options) (*Cache, error) {
	if opts.DBPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = os.TempDir()
		}
		opts.DBPath = filepath.Join(cacheDir, "marc-mcp", "cache.db")
	}

	if opts.TTL == 0 {
		opts.TTL = 24 * time.Hour
	}

	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	if err := os.MkdirAll(filepath.Dir(opts.DBPath), 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	db, err := sql.Open("sqlite", opts.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	opts.Logger.Debug("cache initialized", "path", opts.DBPath, "ttl", opts.TTL)

	return &Cache{
		db:     db,
		logger: opts.Logger,
		ttl:    opts.TTL,
	}, nil
}

func (c *Cache) Close() error {
	return c.db.Close()
}

type MailingList struct {
	Name     string
	Category string
}

func (c *Cache) GetMailingLists() ([]MailingList, bool) {
	cutoff := time.Now().Add(-c.ttl).Unix()

	rows, err := c.db.Query(
		"SELECT name, category FROM mailing_lists WHERE updated_at > ? ORDER BY category, name",
		cutoff,
	)
	if err != nil {
		c.logger.Debug("cache miss: mailing_lists", "error", err)
		return nil, false
	}
	defer rows.Close()

	var lists []MailingList
	for rows.Next() {
		var l MailingList
		if err := rows.Scan(&l.Name, &l.Category); err != nil {
			return nil, false
		}
		lists = append(lists, l)
	}

	if len(lists) == 0 {
		return nil, false
	}

	c.logger.Debug("cache hit: mailing_lists", "count", len(lists))
	return lists, true
}

func (c *Cache) SetMailingLists(lists []MailingList) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	stmt, err := tx.Prepare(
		"INSERT OR REPLACE INTO mailing_lists (name, category, updated_at) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, l := range lists {
		if _, err := stmt.Exec(l.Name, l.Category, now); err != nil {
			return err
		}
	}

	c.logger.Debug("cache set: mailing_lists", "count", len(lists))
	return tx.Commit()
}

type Message struct {
	ID      string
	List    string
	Subject string
	Author  string
	Date    string
}

func (c *Cache) GetMessages(list, month string) ([]Message, bool) {
	cutoff := time.Now().Add(-c.ttl).Unix()

	query := "SELECT id, list, subject, author, date FROM messages WHERE list = ? AND updated_at > ?"
	args := []any{list, cutoff}

	if month != "" {
		query += " AND date LIKE ?"
		// month is YYYYMM, convert to YYYY-MM prefix
		datePrefix := month[:4] + "-" + month[4:] + "%"
		args = append(args, datePrefix)
	}

	query += " ORDER BY date DESC"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		c.logger.Debug("cache miss: messages", "list", list, "error", err)
		return nil, false
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.List, &m.Subject, &m.Author, &m.Date); err != nil {
			return nil, false
		}
		messages = append(messages, m)
	}

	if len(messages) == 0 {
		return nil, false
	}

	c.logger.Debug("cache hit: messages", "list", list, "count", len(messages))
	return messages, true
}

func (c *Cache) SetMessages(messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	stmt, err := tx.Prepare(
		"INSERT OR REPLACE INTO messages (id, list, subject, author, date, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range messages {
		if _, err := stmt.Exec(m.ID, m.List, m.Subject, m.Author, m.Date, now); err != nil {
			return err
		}
	}

	c.logger.Debug("cache set: messages", "count", len(messages))
	return tx.Commit()
}

type MessageContent struct {
	Message
	Body    string
	Headers map[string]string
}

func (c *Cache) GetMessageContent(list, id string) (*MessageContent, bool) {
	cutoff := time.Now().Add(-c.ttl).Unix()

	var m MessageContent
	var headersJSON string

	err := c.db.QueryRow(
		"SELECT id, list, subject, author, date, body, headers FROM message_content WHERE id = ? AND list = ? AND updated_at > ?",
		id, list, cutoff,
	).Scan(&m.ID, &m.List, &m.Subject, &m.Author, &m.Date, &m.Body, &headersJSON)

	if err != nil {
		c.logger.Debug("cache miss: message_content", "id", id, "error", err)
		return nil, false
	}

	if err := json.Unmarshal([]byte(headersJSON), &m.Headers); err != nil {
		return nil, false
	}

	c.logger.Debug("cache hit: message_content", "id", id)
	return &m, true
}

func (c *Cache) SetMessageContent(m *MessageContent) error {
	headersJSON, err := json.Marshal(m.Headers)
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	_, err = c.db.Exec(
		"INSERT OR REPLACE INTO message_content (id, list, subject, author, date, body, headers, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.List, m.Subject, m.Author, m.Date, m.Body, string(headersJSON), now,
	)

	if err == nil {
		c.logger.Debug("cache set: message_content", "id", m.ID)
	}

	return err
}

// SearchMessages performs full-text search across cached messages
func (c *Cache) SearchMessages(query string, list string) ([]Message, error) {
	sqlQuery := `
		SELECT mc.id, mc.list, mc.subject, mc.author, mc.date
		FROM messages_fts fts
		JOIN message_content mc ON fts.rowid = mc.rowid
		WHERE messages_fts MATCH ?
	`
	args := []any{query}

	if list != "" {
		sqlQuery += " AND mc.list = ?"
		args = append(args, list)
	}

	sqlQuery += " ORDER BY rank LIMIT 100"

	rows, err := c.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.List, &m.Subject, &m.Author, &m.Date); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	c.logger.Debug("fts search", "query", query, "results", len(messages))
	return messages, nil
}

// Cleanup removes expired entries
func (c *Cache) Cleanup() error {
	cutoff := time.Now().Add(-c.ttl).Unix()

	tables := []string{"mailing_lists", "messages", "message_content"}
	for _, table := range tables {
		result, err := c.db.Exec("DELETE FROM "+table+" WHERE updated_at < ?", cutoff)
		if err != nil {
			return fmt.Errorf("cleanup %s: %w", table, err)
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			c.logger.Debug("cleanup", "table", table, "deleted", affected)
		}
	}

	return nil
}
