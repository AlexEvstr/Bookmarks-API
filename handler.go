package bookmarks

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxRequestBodySize = 1 << 20

type Handler struct {
	store *Store
	mux   *http.ServeMux
}

type bookmarkInput struct {
	URL   string   `json:"url"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(store *Store) *Handler {
	h := &Handler{
		store: store,
		mux:   http.NewServeMux(),
	}

	h.mux.HandleFunc("POST /bookmarks", h.createBookmark)
	h.mux.HandleFunc("GET /bookmarks", h.listBookmarks)
	h.mux.HandleFunc("GET /bookmarks/{id}", h.getBookmark)
	h.mux.HandleFunc("PUT /bookmarks/{id}", h.updateBookmark)
	h.mux.HandleFunc("DELETE /bookmarks/{id}", h.deleteBookmark)
	h.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
	h.mux.HandleFunc("/bookmarks", methodNotAllowed("GET, HEAD, POST"))
	h.mux.HandleFunc("/bookmarks/{id}", methodNotAllowed("GET, HEAD, PUT, DELETE"))
	h.mux.HandleFunc("/healthz", methodNotAllowed("GET, HEAD"))
	h.mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "route not found")
	})

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) createBookmark(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeBookmarkInput(w, r)
	if !ok {
		return
	}

	bookmark := h.store.Create(input.URL, input.Title, input.Tags)
	writeJSON(w, http.StatusCreated, bookmark)
}

func (h *Handler) listBookmarks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

func (h *Handler) getBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBookmarkID(w, r)
	if !ok {
		return
	}

	bookmark, found := h.store.Get(id)
	if !found {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}

	writeJSON(w, http.StatusOK, bookmark)
}

func (h *Handler) updateBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBookmarkID(w, r)
	if !ok {
		return
	}

	input, ok := decodeBookmarkInput(w, r)
	if !ok {
		return
	}

	bookmark, found := h.store.Update(id, input.URL, input.Title, input.Tags)
	if !found {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}

	writeJSON(w, http.StatusOK, bookmark)
}

func (h *Handler) deleteBookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := parseBookmarkID(w, r)
	if !ok {
		return
	}

	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeBookmarkInput(w http.ResponseWriter, r *http.Request) (bookmarkInput, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input bookmarkInput
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return bookmarkInput{}, false
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return bookmarkInput{}, false
	}

	input, message := normalizeBookmarkInput(input)
	if message != "" {
		writeError(w, http.StatusBadRequest, message)
		return bookmarkInput{}, false
	}

	return input, true
}

func normalizeBookmarkInput(input bookmarkInput) (bookmarkInput, string) {
	input.URL = strings.TrimSpace(input.URL)
	input.Title = strings.TrimSpace(input.Title)
	if input.URL == "" || input.Title == "" {
		return bookmarkInput{}, "url and title are required"
	}

	parsed, err := url.Parse(input.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return bookmarkInput{}, "url must be an absolute http or https URL with a host"
	}
	for _, tag := range input.Tags {
		if strings.TrimSpace(tag) == "" {
			return bookmarkInput{}, "tags must not contain empty values"
		}
	}
	return input, ""
}

func parseBookmarkID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid bookmark id")
		return 0, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
