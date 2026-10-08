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

func TestHandlerRouteErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
	}{
		{name: "unsupported collection method", method: http.MethodPatch, path: "/bookmarks", wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET, HEAD, POST"},
		{name: "unsupported item method", method: http.MethodPost, path: "/bookmarks/1", wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET, HEAD, PUT, DELETE"},
		{name: "unknown route", method: http.MethodGet, path: "/unknown", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewHandler(NewStore()).ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if got := response.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
			var apiError errorResponse
			decodeResponseJSON(t, response, &apiError)
			if apiError.Error == "" {
				t.Error("error response has an empty error message")
			}
		})
	}
}

func TestHandlerHealthz(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(NewStore()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		Status string `json:"status"`
	}
	decodeResponseJSON(t, response, &body)
	if body.Status != "ok" {
		t.Errorf("status body = %q, want ok", body.Status)
	}
}

func TestHandlerBookmarkInputValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantURL    string
		wantTitle  string
	}{
		{name: "trim URL and title", body: `{"url":"  https://go.dev/docs  ","title":"  Go docs  "}`, wantStatus: http.StatusCreated, wantURL: "https://go.dev/docs", wantTitle: "Go docs"},
		{name: "URL without scheme", body: `{"url":"go.dev/docs","title":"Go"}`, wantStatus: http.StatusBadRequest},
		{name: "unsupported scheme", body: `{"url":"ftp://go.dev/docs","title":"Go"}`, wantStatus: http.StatusBadRequest},
		{name: "URL without host", body: `{"url":"https:///docs","title":"Go"}`, wantStatus: http.StatusBadRequest},
		{name: "URL with port but no hostname", body: `{"url":"https://:8080/docs","title":"Go"}`, wantStatus: http.StatusBadRequest},
		{name: "valid HTTP URL", body: `{"url":"http://example.com","title":"Example"}`, wantStatus: http.StatusCreated, wantURL: "http://example.com", wantTitle: "Example"},
		{name: "valid HTTPS URL", body: `{"url":"https://example.com/path?q=1","title":"Example"}`, wantStatus: http.StatusCreated, wantURL: "https://example.com/path?q=1", wantTitle: "Example"},
		{name: "empty tags allowed", body: `{"url":"https://example.com","title":"Example","tags":[]}`, wantStatus: http.StatusCreated, wantURL: "https://example.com", wantTitle: "Example"},
		{name: "empty tag rejected", body: `{"url":"https://example.com","title":"Example","tags":[""]}`, wantStatus: http.StatusBadRequest},
		{name: "blank tag rejected", body: `{"url":"https://example.com","title":"Example","tags":["go","  "]}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			response := httptest.NewRecorder()
			NewHandler(store).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/bookmarks", strings.NewReader(tt.body)))

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantStatus == http.StatusBadRequest {
				var apiError errorResponse
				decodeResponseJSON(t, response, &apiError)
				if apiError.Error == "" {
					t.Error("error response has an empty error message")
				}
				if got := store.List(); len(got) != 0 {
					t.Errorf("invalid input saved %d bookmarks, want none", len(got))
				}
				return
			}

			var created Bookmark
			decodeResponseJSON(t, response, &created)
			if created.URL != tt.wantURL || created.Title != tt.wantTitle {
				t.Errorf("created URL/title = %q/%q, want %q/%q", created.URL, created.Title, tt.wantURL, tt.wantTitle)
			}
			stored, found := store.Get(created.ID)
			if !found || stored.URL != tt.wantURL || stored.Title != tt.wantTitle {
				t.Errorf("stored bookmark = %+v, found = %v", stored, found)
			}
		})
	}
}

func TestHandlerUpdateValidatesBeforeSaving(t *testing.T) {
	store := NewStore()
	created := store.Create("https://example.com/old", "Old", nil)
	handler := NewHandler(store)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantURL    string
		wantTitle  string
	}{
		{name: "invalid URL preserves bookmark", body: `{"url":"ftp://example.com","title":"New"}`, wantStatus: http.StatusBadRequest, wantURL: created.URL, wantTitle: created.Title},
		{name: "trim before update", body: `{"url":"  http://example.com/new  ","title":"  New  "}`, wantStatus: http.StatusOK, wantURL: "http://example.com/new", wantTitle: "New"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/bookmarks/1", strings.NewReader(tt.body)))
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			stored, found := store.Get(created.ID)
			if !found || stored.URL != tt.wantURL || stored.Title != tt.wantTitle {
				t.Errorf("stored URL/title = %q/%q, found = %v; want %q/%q", stored.URL, stored.Title, found, tt.wantURL, tt.wantTitle)
			}
		})
	}
}

func decodeResponseJSON(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response JSON: %v; body = %s", err, response.Body.String())
	}
}
