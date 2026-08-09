package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice","unexpected":true}`))
	recorder := httptest.NewRecorder()
	var payload struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(recorder, req, &payload); err == nil {
		t.Fatal("expected an error")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestDecodeJSONRejectsMultipleDocuments(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"} {"name":"bob"}`))
	recorder := httptest.NewRecorder()
	var payload struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(recorder, req, &payload); err == nil {
		t.Fatal("expected an error")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestReadBinaryBodyRejectsOversizePayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(strings.Repeat("x", maxBinaryPayload+1)))
	recorder := httptest.NewRecorder()
	if _, err := readBinaryBody(recorder, req); err == nil {
		t.Fatal("expected an error")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}
