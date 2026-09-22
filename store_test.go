package bookmarks

import (
	"fmt"
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

func TestStoreListEmpty(t *testing.T) {
	bookmarks := NewStore().List()

	if bookmarks == nil {
		t.Errorf("List() = nil, want empty slice")
	}

	if len(bookmarks) != 0 {
		t.Errorf("List() = %v, want empty slice", bookmarks)
	}
}

func TestStoreListReturnsBookmarksByID(t *testing.T) {
	store := NewStore()
	const count = 20
	for i := 0; i < count; i++ {
		store.Create(
			fmt.Sprintf("https://example.com/%d", i),
			fmt.Sprintf("Example %d", i),
			[]string{fmt.Sprintf("tag%d", i)},
		)
	}

	bookmarks := store.List()

	if len(bookmarks) != count {
		t.Fatalf("List() = %v, want %d bookmarks", bookmarks, count)
	}

	for i := range bookmarks {
		if bookmarks[i].ID != int64(i+1) {
			t.Errorf("List() bookmark at index %d has ID = %v, want %v", i, bookmarks[i].ID, int64(i+1))
		}
	}
}

func TestStoreListCopiesTags(t *testing.T) {
	store := NewStore()
	_ = store.Create("https://example.com", "Example", []string{"go"})
	bookmarks := store.List()

	if len(bookmarks) != 1 {
		t.Fatalf("List() = %v, want 1 bookmark", bookmarks)
	}

	// Modify the tags of the returned bookmark
	bookmarks[0].Tags[0] = "modified"

	// Retrieve the bookmarks again
	bookmarks2 := store.List()

	if bookmarks2[0].Tags[0] != "go" {
		t.Errorf("List() did not copy tags, got %v, want %v", bookmarks2[0].Tags[0], "go")
	}
}
