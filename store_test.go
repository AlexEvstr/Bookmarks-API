package bookmarks

import (
	"slices"
	"testing"
)

func TestStoreCreate(t *testing.T) {
	store := NewStore()

	tests := []struct {
		name   string
		url    string
		title  string
		tags   []string
		wantID int64
	}{
		{
			name:   "one tag",
			url:    "https://example.com",
			title:  "Example",
			tags:   []string{"tag1"},
			wantID: 1,
		},
		{
			name:   "multiple tags",
			url:    "https://example.org",
			title:  "Example Org",
			tags:   []string{"tag2", "tag3"},
			wantID: 2,
		},
		{
			name:   "nil tags",
			url:    "https://example.net",
			title:  "Example Net",
			tags:   nil,
			wantID: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.Create(tt.url, tt.title, tt.tags)

			if got.ID != tt.wantID {
				t.Errorf("Create() ID = %v, want %v", got.ID, tt.wantID)
			}
			if got.URL != tt.url {
				t.Errorf("Create() URL = %v, want %v", got.URL, tt.url)
			}
			if got.Title != tt.title {
				t.Errorf("Create() Title = %v, want %v", got.Title, tt.title)
			}
			if !slices.Equal(got.Tags, tt.tags) {
				t.Errorf("Create() Tags = %v, want %v", got.Tags, tt.tags)
			}
			if got.CreatedAt.IsZero() {
				t.Errorf("Create() CreatedAt is zero")
			}
			if got.UpdatedAt.IsZero() {
				t.Errorf("Create() UpdatedAt is zero")
			}
			if !got.UpdatedAt.Equal(got.CreatedAt) {
				t.Errorf("Create() UpdatedAt = %v, want %v", got.UpdatedAt, got.CreatedAt)
			}
		})
	}
}

func TestStoreCreateCopiesTags(t *testing.T) {
	store := NewStore()
	tags := []string{"tag1", "tag2"}
	bookmark := store.Create("https://example.com", "Example", tags)

	// Modify the original tags slice
	tags[0] = "modified"

	want := []string{"tag1", "tag2"}

	if !slices.Equal(bookmark.Tags, want) {
		t.Errorf("Create() Tags = %v, want %v", bookmark.Tags, want)
	}
}
