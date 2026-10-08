package bookmarks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestHandlerCreateAndGetBookmark(t *testing.T) {
	handler := NewHandler(NewStore())

	createRequest := httptest.NewRequest(http.MethodPost, "/bookmarks", strings.NewReader(`{
		"url":"https://go.dev",
		"title":"Go",
		"tags":["go","docs"]
	}`))
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST /bookmarks status = %d, want %d; body = %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}

	var created Bookmark
	decodeResponseJSON(t, createResponse, &created)
	if created.ID != 1 || created.URL != "https://go.dev" || created.Title != "Go" {
		t.Errorf("POST /bookmarks body = %+v, want created bookmark", created)
	}
	if !slices.Equal(created.Tags, []string{"go", "docs"}) {
		t.Errorf("POST /bookmarks tags = %v, want %v", created.Tags, []string{"go", "docs"})
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Errorf("POST /bookmarks timestamps = %v, %v, want non-zero", created.CreatedAt, created.UpdatedAt)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/bookmarks/1", nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /bookmarks/1 status = %d, want %d", getResponse.Code, http.StatusOK)
	}
	var got Bookmark
	decodeResponseJSON(t, getResponse, &got)
	if got.ID != created.ID || got.URL != created.URL || got.Title != created.Title || !slices.Equal(got.Tags, created.Tags) {
		t.Errorf("GET /bookmarks/1 body = %+v, want %+v", got, created)
	}
}

func TestHandlerListBookmarks(t *testing.T) {
	store := NewStore()
	store.Create("https://example.com/1", "First", nil)
	store.Create("https://example.com/2", "Second", []string{"two"})
	handler := NewHandler(store)

	request := httptest.NewRequest(http.MethodGet, "/bookmarks", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /bookmarks status = %d, want %d", response.Code, http.StatusOK)
	}
	var bookmarks []Bookmark
	decodeResponseJSON(t, response, &bookmarks)
	if len(bookmarks) != 2 {
		t.Fatalf("GET /bookmarks returned %d bookmarks, want 2", len(bookmarks))
	}
	if bookmarks[0].ID != 1 || bookmarks[1].ID != 2 {
		t.Errorf("GET /bookmarks IDs = %d, %d, want 1, 2", bookmarks[0].ID, bookmarks[1].ID)
	}
}

func TestHandlerListBookmarksEmpty(t *testing.T) {
	handler := NewHandler(NewStore())
	request := httptest.NewRequest(http.MethodGet, "/bookmarks", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /bookmarks status = %d, want %d", response.Code, http.StatusOK)
	}
	var bookmarks []Bookmark
	decodeResponseJSON(t, response, &bookmarks)
	if bookmarks == nil || len(bookmarks) != 0 {
		t.Errorf("GET /bookmarks body = %v, want empty JSON array", bookmarks)
	}
}

func TestHandlerUpdateBookmark(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", []string{"old"})
	handler := NewHandler(store)
	request := httptest.NewRequest(http.MethodPut, "/bookmarks/1", strings.NewReader(`{
		"url":"https://go.dev",
		"title":"Go",
		"tags":["go"]
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("PUT /bookmarks/1 status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var updated Bookmark
	decodeResponseJSON(t, response, &updated)
	if updated.ID != created.ID || updated.URL != "https://go.dev" || updated.Title != "Go" || !slices.Equal(updated.Tags, []string{"go"}) {
		t.Errorf("PUT /bookmarks/1 body = %+v, want updated bookmark", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("PUT /bookmarks/1 timestamps = %v, %v; created bookmark = %v, %v", updated.CreatedAt, updated.UpdatedAt, created.CreatedAt, created.UpdatedAt)
	}
}

func TestHandlerDeleteBookmark(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com", "Example", nil)
	handler := NewHandler(store)
	request := httptest.NewRequest(http.MethodDelete, "/bookmarks/1", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE /bookmarks/1 status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Body.Len() != 0 {
		t.Errorf("DELETE /bookmarks/1 body = %q, want empty", response.Body.String())
	}
	if _, ok := store.Get(created.ID); ok {
		t.Error("bookmark still exists after DELETE")
	}
}

func TestHandlerErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		setup      func(*Store)
		wantStatus int
	}{
		{name: "malformed JSON", method: http.MethodPost, path: "/bookmarks", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "extra JSON value", method: http.MethodPost, path: "/bookmarks", body: `{"url":"https://go.dev","title":"Go"} {}`, wantStatus: http.StatusBadRequest},
		{name: "unknown JSON field", method: http.MethodPost, path: "/bookmarks", body: `{"url":"https://go.dev","title":"Go","unknown":true}`, wantStatus: http.StatusBadRequest},
		{name: "missing URL", method: http.MethodPost, path: "/bookmarks", body: `{"title":"Go"}`, wantStatus: http.StatusBadRequest},
		{name: "blank title", method: http.MethodPost, path: "/bookmarks", body: `{"url":"https://go.dev","title":"  "}`, wantStatus: http.StatusBadRequest},
		{name: "invalid ID", method: http.MethodGet, path: "/bookmarks/nope", wantStatus: http.StatusBadRequest},
		{name: "non-positive ID", method: http.MethodGet, path: "/bookmarks/0", wantStatus: http.StatusBadRequest},
		{name: "missing bookmark", method: http.MethodGet, path: "/bookmarks/999", wantStatus: http.StatusNotFound},
		{name: "update missing bookmark", method: http.MethodPut, path: "/bookmarks/999", body: `{"url":"https://go.dev","title":"Go"}`, wantStatus: http.StatusNotFound},
		{name: "delete missing bookmark", method: http.MethodDelete, path: "/bookmarks/999", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			handler := NewHandler(store)
			request := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d, want %d; body = %s", tt.method, tt.path, response.Code, tt.wantStatus, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
			var apiError errorResponse
			decodeResponseJSON(t, response, &apiError)
			if apiError.Error == "" {
				t.Error("error response has an empty error message")
			}
		})
	}
}

func decodeResponseJSON(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response JSON: %v; body = %s", err, response.Body.String())
	}
}
