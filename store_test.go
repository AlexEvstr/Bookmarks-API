package bookmarks

import (
	"fmt"
	"reflect"
	"slices"
	"sync"
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

func TestStoreCreateCopiesReturnedTags(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", []string{"go"})

	created.Tags[0] = "modified"

	stored, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("Get() after Create() = _, false, want true")
	}
	if !slices.Equal(stored.Tags, []string{"go"}) {
		t.Errorf("stored Tags = %v, want %v", stored.Tags, []string{"go"})
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

func TestStoreGetExisting(t *testing.T) {
	store := NewStore()
	bookmark := store.Create("https://example.com", "Example", []string{"go"})

	got, ok := store.Get(bookmark.ID)
	if !ok {
		t.Fatalf("Get() = _, false, want true")
	}

	if got.ID != bookmark.ID {
		t.Errorf("Get() ID = %v, want %v", got.ID, bookmark.ID)
	}
	if got.URL != bookmark.URL {
		t.Errorf("Get() URL = %v, want %v", got.URL, bookmark.URL)
	}
	if got.Title != bookmark.Title {
		t.Errorf("Get() Title = %v, want %v", got.Title, bookmark.Title)
	}
	if !slices.Equal(got.Tags, bookmark.Tags) {
		t.Errorf("Get() Tags = %v, want %v", got.Tags, bookmark.Tags)
	}
	if !got.CreatedAt.Equal(bookmark.CreatedAt) {
		t.Errorf("Get() CreatedAt = %v, want %v", got.CreatedAt, bookmark.CreatedAt)
	}
	if !got.UpdatedAt.Equal(bookmark.UpdatedAt) {
		t.Errorf("Get() UpdatedAt = %v, want %v", got.UpdatedAt, bookmark.UpdatedAt)
	}
}

func TestStoreGetMissing(t *testing.T) {
	store := NewStore()
	got, ok := store.Get(999)
	if ok {
		t.Fatalf("Get() = %v, true, want false", got)
	}
	if !reflect.DeepEqual(got, Bookmark{}) {
		t.Errorf("Get() bookmark = %v, want zero Bookmark", got)
	}
}

func TestStoreGetCopiesTags(t *testing.T) {
	store := NewStore()
	bookmark := store.Create("https://example.com", "go", []string{"go"})

	got, ok := store.Get(bookmark.ID)
	if !ok {
		t.Fatalf("Get() = _, false, want true")
	}

	// Modify the tags of the returned bookmark
	got.Tags[0] = "modified"

	// Retrieve the bookmark again
	got2, ok := store.Get(bookmark.ID)
	if !ok {
		t.Fatalf("Get() = _, false, want true")
	}

	if got2.Tags[0] != "go" {
		t.Errorf("Get() did not copy tags, got %v, want %v", got2.Tags[0], "go")
	}
}

func TestStoreUpdateExisting(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", []string{"old"})

	updated, ok := store.Update(created.ID, "https://go.dev", "Go", []string{"go", "docs"})
	if !ok {
		t.Fatal("Update() = _, false, want true")
	}

	if updated.ID != created.ID {
		t.Errorf("Update() ID = %d, want %d", updated.ID, created.ID)
	}
	if updated.URL != "https://go.dev" {
		t.Errorf("Update() URL = %q, want %q", updated.URL, "https://go.dev")
	}
	if updated.Title != "Go" {
		t.Errorf("Update() Title = %q, want %q", updated.Title, "Go")
	}
	if !slices.Equal(updated.Tags, []string{"go", "docs"}) {
		t.Errorf("Update() Tags = %v, want %v", updated.Tags, []string{"go", "docs"})
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("Update() CreatedAt = %v, want %v", updated.CreatedAt, created.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("Update() UpdatedAt = %v, want a time after %v", updated.UpdatedAt, created.UpdatedAt)
	}

	stored, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("Get() after Update() = _, false, want true")
	}
	if stored.URL != updated.URL || stored.Title != updated.Title || !slices.Equal(stored.Tags, updated.Tags) {
		t.Errorf("Get() after Update() = %v, want updated bookmark %v", stored, updated)
	}
}

func TestStoreUpdateMissing(t *testing.T) {
	store := NewStore()

	got, ok := store.Update(999, "https://go.dev", "Go", []string{"go"})
	if ok {
		t.Fatalf("Update() = %v, true, want false", got)
	}
	if !reflect.DeepEqual(got, Bookmark{}) {
		t.Errorf("Update() bookmark = %v, want zero Bookmark", got)
	}
}

func TestStoreUpdateCopiesInputTags(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", nil)
	tags := []string{"go"}

	_, ok := store.Update(created.ID, "https://go.dev", "Go", tags)
	if !ok {
		t.Fatal("Update() = _, false, want true")
	}
	tags[0] = "modified"

	stored, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("Get() after Update() = _, false, want true")
	}
	if !slices.Equal(stored.Tags, []string{"go"}) {
		t.Errorf("stored Tags = %v, want %v", stored.Tags, []string{"go"})
	}
}

func TestStoreUpdateCopiesReturnedTags(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", nil)

	updated, ok := store.Update(created.ID, "https://go.dev", "Go", []string{"go"})
	if !ok {
		t.Fatal("Update() = _, false, want true")
	}
	updated.Tags[0] = "modified"

	stored, ok := store.Get(created.ID)
	if !ok {
		t.Fatal("Get() after Update() = _, false, want true")
	}
	if !slices.Equal(stored.Tags, []string{"go"}) {
		t.Errorf("stored Tags = %v, want %v", stored.Tags, []string{"go"})
	}
}

func TestStoreDeleteExisting(t *testing.T) {
	store := NewStore()
	first := store.Create("https://example.com/1", "First", nil)
	second := store.Create("https://example.com/2", "Second", nil)

	if deleted := store.Delete(first.ID); !deleted {
		t.Fatal("Delete() = false, want true")
	}
	if got, ok := store.Get(first.ID); ok {
		t.Errorf("Get() deleted bookmark = %v, true, want false", got)
	}
	if got, ok := store.Get(second.ID); !ok || got.ID != second.ID {
		t.Errorf("Get() retained bookmark = %v, %v, want ID %d and true", got, ok, second.ID)
	}

	third := store.Create("https://example.com/3", "Third", nil)
	if third.ID != 3 {
		t.Errorf("Create() after Delete() ID = %d, want 3", third.ID)
	}
}

func TestStoreDeleteMissing(t *testing.T) {
	store := NewStore()

	if deleted := store.Delete(999); deleted {
		t.Error("Delete() = true, want false")
	}
}

func TestStoreConcurrentCreate(t *testing.T) {
	store := NewStore()
	const count = 100

	var waitGroup sync.WaitGroup
	for i := 0; i < count; i++ {
		waitGroup.Add(1)
		go func(i int) {
			defer waitGroup.Done()
			store.Create(
				fmt.Sprintf("https://example.com/%d", i),
				fmt.Sprintf("Example %d", i),
				nil,
			)
		}(i)
	}
	waitGroup.Wait()

	bookmarks := store.List()
	if len(bookmarks) != count {
		t.Fatalf("List() returned %d bookmarks, want %d", len(bookmarks), count)
	}
	for i, bookmark := range bookmarks {
		wantID := int64(i + 1)
		if bookmark.ID != wantID {
			t.Errorf("bookmark at index %d has ID %d, want %d", i, bookmark.ID, wantID)
		}
	}
}
