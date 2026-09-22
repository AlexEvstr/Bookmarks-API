package bookmarks

import (
	"slices"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu        sync.RWMutex
	bookmarks map[int64]Bookmark
	nextID    int64
}

func NewStore() *Store {
	return &Store{
		bookmarks: make(map[int64]Bookmark),
		nextID:    1,
	}
}

func (s *Store) Create(url, title string, tags []string) Bookmark {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	bookmark := Bookmark{
		ID:        s.nextID,
		URL:       url,
		Title:     title,
		Tags:      slices.Clone(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.bookmarks[bookmark.ID] = bookmark
	s.nextID++
	return bookmark
}

func (s *Store) List() []Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bookmarks := make([]Bookmark, 0, len(s.bookmarks))
	for _, bookmark := range s.bookmarks {
		bookmark.Tags = slices.Clone(bookmark.Tags)
		bookmarks = append(bookmarks, bookmark)
	}
	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].ID < bookmarks[j].ID
	})
	return bookmarks
}

func (s *Store) Get(id int64) (Bookmark, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bookmark, exists := s.bookmarks[id]

	if !exists {
		return Bookmark{}, false
	}
	bookmark.Tags = slices.Clone(bookmark.Tags)
	return bookmark, true
}

func (s *Store) Update(id int64, url, title string, tags []string) (Bookmark, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bookmark, exists := s.bookmarks[id]
	if !exists {
		return Bookmark{}, false
	}

	bookmark.URL = url
	bookmark.Title = title
	bookmark.Tags = slices.Clone(tags)
	bookmark.UpdatedAt = time.Now()
	s.bookmarks[id] = bookmark
	return bookmark, true
}

func (s *Store) Delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.bookmarks[id]; !exists {
		return false
	}
	delete(s.bookmarks, id)
	return true
}
