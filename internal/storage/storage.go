package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	ErrNoGroup   = errors.New("group is not registered")
	ErrNoChannel = errors.New("channel is not in group")
)

type Channel struct {
	ID       int64  `json:"id"`
	Username string `json:"username,omitempty"`
	Title    string `json:"title"`
}

type Group struct {
	ID                int64             `json:"id"`
	Title             string            `json:"title"`
	ForbiddenChannels map[int64]Channel `json:"forbidden_channels"`
}

type Store struct {
	db *sql.DB

	mu sync.Mutex
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) init() error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS groups (
			id    INTEGER PRIMARY KEY,
			title TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS forbidden_channels (
			group_id   INTEGER NOT NULL,
			channel_id INTEGER NOT NULL,
			username   TEXT NOT NULL DEFAULT '',
			title      TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (group_id, channel_id),
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UpsertGroup(id int64, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var existing string
	err := s.db.QueryRow(`SELECT title FROM groups WHERE id = ?`, id).Scan(&existing)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && (title == "" || existing == title) {
		return nil
	}

	_, err = s.db.Exec(
		`INSERT INTO groups (id, title) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET title = excluded.title`,
		id, title,
	)
	return err
}

func (s *Store) GroupsSnapshot() []Group {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT id, title FROM groups ORDER BY title`)
	if err != nil {
		return nil
	}

	type row struct {
		id    int64
		title string
	}
	var found []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.title); err != nil {
			rows.Close()
			return nil
		}
		found = append(found, r)
	}
	rows.Close()

	result := make([]Group, 0, len(found))
	for _, r := range found {
		g := Group{ID: r.id, Title: r.title, ForbiddenChannels: s.channelsFor(r.id)}
		result = append(result, g)
	}
	return result
}

func (s *Store) channelsFor(groupID int64) map[int64]Channel {
	rows, err := s.db.Query(
		`SELECT channel_id, username, title FROM forbidden_channels WHERE group_id = ? ORDER BY title`,
		groupID,
	)
	if err != nil {
		return map[int64]Channel{}
	}
	defer rows.Close()

	result := make(map[int64]Channel)
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.Username, &ch.Title); err != nil {
			return result
		}
		result[ch.ID] = ch
	}
	return result
}

func (s *Store) GetForbiddenChannels(groupID int64) []Channel {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT channel_id, username, title FROM forbidden_channels WHERE group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	channels := make([]Channel, 0)
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.Username, &ch.Title); err != nil {
			return channels
		}
		channels = append(channels, ch)
	}
	sort.Slice(channels, func(i, j int) bool {
		return DisplayChannel(channels[i]) < DisplayChannel(channels[j])
	})
	return channels
}

func (s *Store) AddForbiddenChannel(groupID int64, channel Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.groupExists(groupID) {
		return ErrNoGroup
	}

	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO forbidden_channels (group_id, channel_id, username, title)
		 VALUES (?, ?, ?, ?)`,
		groupID, channel.ID, channel.Username, channel.Title,
	)
	return err
}

func (s *Store) RemoveForbiddenChannel(groupID, channelID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.groupExists(groupID) {
		return ErrNoGroup
	}

	res, err := s.db.Exec(
		`DELETE FROM forbidden_channels WHERE group_id = ? AND channel_id = ?`,
		groupID, channelID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNoChannel
	}
	return nil
}

func (s *Store) IsForbidden(groupID, channelID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM forbidden_channels WHERE group_id = ? AND channel_id = ?)`,
		groupID, channelID,
	).Scan(&exists)
	return err == nil && exists
}

func (s *Store) groupExists(groupID int64) bool {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM groups WHERE id = ?)`, groupID).Scan(&exists)
	return err == nil && exists
}

func DisplayChannel(channel Channel) string {
	if channel.Username != "" {
		return "@" + strings.TrimPrefix(channel.Username, "@")
	}
	if channel.Title != "" {
		return channel.Title
	}
	return fmt.Sprintf("%d", channel.ID)
}
