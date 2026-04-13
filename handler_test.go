package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlePreviewTextReturnsContentWithoutConsumingOneTime(t *testing.T) {
	app := NewApp("", "")

	item, err := app.store.AddText("hello world")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}
	item.OneTimeUse = true

	req := httptest.NewRequest(http.MethodGet, "/preview/"+item.Token, nil)
	req.SetPathValue("token", item.Token)
	rec := httptest.NewRecorder()
	app.handlePreview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handlePreview() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		Preview string `json:"preview"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if payload.Preview != "hello world" {
		t.Fatalf("preview snippet = %q, want %q", payload.Preview, "hello world")
	}

	recvReq := httptest.NewRequest(http.MethodGet, "/recv/"+item.Token, nil)
	recvReq.SetPathValue("token", item.Token)
	recvRec := httptest.NewRecorder()
	app.handleRecv(recvRec, recvReq)

	if recvRec.Code != http.StatusOK {
		t.Fatalf("handleRecv() after preview status = %d, want %d", recvRec.Code, http.StatusOK)
	}
	if _, ok := app.store.Get(item.Token); ok {
		t.Fatal("expected one-time text token to be removed only after the real receive")
	}
}

func TestHandleRecvHeadDoesNotConsumeOneTimeFile(t *testing.T) {
	app := NewApp("", "")

	item, err := app.store.AddFile("demo.txt", []byte("abcdef"), 6)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}
	item.OneTimeUse = true

	req := httptest.NewRequest(http.MethodHead, "/recv/"+item.Token, nil)
	req.SetPathValue("token", item.Token)
	rec := httptest.NewRecorder()
	app.handleRecv(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD /recv status = %d, want %d", rec.Code, http.StatusOK)
	}

	if _, found, unavailable := app.store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("BeginDownload() after HEAD = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}
	app.store.FinishDownload(item.Token, DownloadReleased, "test_peer")
}

func TestHandleRecvOneTimeRangeResumeConsumesOnlyAfterEOF(t *testing.T) {
	app := NewApp("", "")

	item, err := app.store.AddFile("demo.txt", []byte("abcdef"), 6)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}
	item.OneTimeUse = true

	firstReq := httptest.NewRequest(http.MethodGet, "/recv/"+item.Token, nil)
	firstReq.Header.Set("Range", "bytes=0-1")
	firstReq.SetPathValue("token", item.Token)
	firstRec := httptest.NewRecorder()
	app.handleRecv(firstRec, firstReq)

	if firstRec.Code != http.StatusPartialContent {
		t.Fatalf("first range status = %d, want %d", firstRec.Code, http.StatusPartialContent)
	}
	if got := firstRec.Body.String(); got != "ab" {
		t.Fatalf("first range body = %q, want %q", got, "ab")
	}
	if _, ok := app.store.Get(item.Token); !ok {
		t.Fatal("expected one-time token to remain after a partial range response")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/recv/"+item.Token, nil)
	secondReq.Header.Set("Range", "bytes=2-")
	secondReq.SetPathValue("token", item.Token)
	secondRec := httptest.NewRecorder()
	app.handleRecv(secondRec, secondReq)

	if secondRec.Code != http.StatusPartialContent {
		t.Fatalf("resume range status = %d, want %d", secondRec.Code, http.StatusPartialContent)
	}
	if got := secondRec.Body.String(); got != "cdef" {
		t.Fatalf("resume range body = %q, want %q", got, "cdef")
	}
	if _, ok := app.store.Get(item.Token); ok {
		t.Fatal("expected one-time token to be removed after the resumed download reached EOF")
	}
}
