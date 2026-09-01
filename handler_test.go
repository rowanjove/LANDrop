package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleRecvUsedOneTimeTokenReturnsGone(t *testing.T) {
	app := NewApp("", "")
	item, err := app.store.AddFile("demo.txt", []byte("abcdef"), 6)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}
	item.OneTimeUse = true

	first := httptest.NewRequest(http.MethodGet, "/recv/"+item.Token, nil)
	first.SetPathValue("token", item.Token)
	firstRec := httptest.NewRecorder()
	app.handleRecv(firstRec, first)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first receive status = %d, want %d", firstRec.Code, http.StatusOK)
	}

	second := httptest.NewRequest(http.MethodGet, "/recv/"+item.Token, nil)
	second.SetPathValue("token", item.Token)
	secondRec := httptest.NewRecorder()
	app.handleRecv(secondRec, second)
	if secondRec.Code != http.StatusGone {
		t.Fatalf("second receive status = %d, want %d", secondRec.Code, http.StatusGone)
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(secondRec.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if payload.Code != "TOKEN_USED" {
		t.Fatalf("second receive code = %q, want TOKEN_USED", payload.Code)
	}
}

func TestHandleShareInfoDoesNotConsumeOneTimeToken(t *testing.T) {
	app := NewApp("", "")
	item, err := app.store.AddText("shared text")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}
	item.OneTimeUse = true

	req := httptest.NewRequest(http.MethodGet, "/api/v2/share/"+item.Token, nil)
	req.SetPathValue("token", item.Token)
	rec := httptest.NewRecorder()
	app.handleShareInfo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("share info status = %d, want %d", rec.Code, http.StatusOK)
	}
	if _, found, unavailable := app.store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("share info consumed token: found=%v unavailable=%v", found, unavailable)
	}
	app.store.FinishDownload(item.Token, DownloadReleased, "test")
}

func TestHandleSendFileMultiCreatesIndependentItemsAndZip(t *testing.T) {
	app := NewApp("", "")
	defer app.store.Cleanup()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for _, file := range []struct {
		name string
		data string
	}{
		{name: "one.txt", data: "one"},
		{name: "two.txt", data: "two"},
	} {
		part, err := writer.CreateFormFile("file", file.name)
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		_, _ = part.Write([]byte(file.data))
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Writer.Close() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/send/file", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	app.handleSendFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("multi upload status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var upload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&upload); err != nil {
		t.Fatalf("upload response decode error = %v", err)
	}

	shareReq := httptest.NewRequest(http.MethodGet, "/api/v2/share/"+upload.Token, nil)
	shareReq.SetPathValue("token", upload.Token)
	shareRec := httptest.NewRecorder()
	app.handleShareInfo(shareRec, shareReq)
	if shareRec.Code != http.StatusOK {
		t.Fatalf("share info status = %d, want %d", shareRec.Code, http.StatusOK)
	}
	var share struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(shareRec.Body).Decode(&share); err != nil {
		t.Fatalf("share response decode error = %v", err)
	}
	if len(share.Items) != 2 {
		t.Fatalf("share item count = %d, want 2", len(share.Items))
	}

	itemReq := httptest.NewRequest(http.MethodGet, "/api/v2/share/"+upload.Token+"/items/"+share.Items[0].ID, nil)
	itemReq.SetPathValue("token", upload.Token)
	itemReq.SetPathValue("itemID", share.Items[0].ID)
	itemRec := httptest.NewRecorder()
	app.handleRecv(itemRec, itemReq)
	if itemRec.Code != http.StatusOK || itemRec.Body.String() != "one" {
		t.Fatalf("item download = status %d body %q, want 200/one", itemRec.Code, itemRec.Body.String())
	}

	zipReq := httptest.NewRequest(http.MethodGet, "/recv/"+upload.Token, nil)
	zipReq.SetPathValue("token", upload.Token)
	zipRec := httptest.NewRecorder()
	app.handleRecv(zipRec, zipReq)
	if zipRec.Code != http.StatusOK {
		t.Fatalf("zip download status = %d, want %d", zipRec.Code, http.StatusOK)
	}
	archive, err := zip.NewReader(bytes.NewReader(zipRec.Body.Bytes()), int64(zipRec.Body.Len()))
	if err != nil {
		t.Fatalf("zip reader error = %v", err)
	}
	if len(archive.File) != 2 {
		t.Fatalf("zip entry count = %d, want 2", len(archive.File))
	}
	contents := make(map[string]string)
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry error = %v", err)
		}
		data, _ := io.ReadAll(reader)
		_ = reader.Close()
		contents[file.Name] = string(data)
	}
	if contents["one.txt"] != "one" || contents["two.txt"] != "two" {
		t.Fatalf("zip contents = %#v, want both uploaded files", contents)
	}
}

func TestHandleIndexServesSharedResponsivePageForAllUserAgents(t *testing.T) {
	app := NewApp("", "")

	render := func(ua string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", ua)
		rec := httptest.NewRecorder()
		app.handleIndex(rec, req)
		return rec
	}

	desktop := render("Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	mobile := render("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1")

	if desktop.Code != http.StatusOK || mobile.Code != http.StatusOK {
		t.Fatalf("handleIndex() status desktop=%d mobile=%d, want %d", desktop.Code, mobile.Code, http.StatusOK)
	}
	if desktop.Body.String() != mobile.Body.String() {
		t.Fatal("expected all devices to receive the same responsive HTML")
	}
	if got := desktop.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store, max-age=0")
	}
	if !strings.Contains(desktop.Body.String(), "LANDrop") {
		t.Fatal("expected shared page markup to include LANDrop title")
	}
}

func TestHandleSendTextRejectsOversizedContent(t *testing.T) {
	app := NewApp("", "")

	body, err := json.Marshal(map[string]string{
		"content": strings.Repeat("a", maxTextSize+1),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/send/text", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	app.handleSendText(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("handleSendText() status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	var payload struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if payload.Code != "TEXT_TOO_LARGE" {
		t.Fatalf("code = %q, want %q", payload.Code, "TEXT_TOO_LARGE")
	}
	if !strings.Contains(payload.Error, "Text too large") {
		t.Fatalf("error = %q, want readable size guidance", payload.Error)
	}
}

func TestApiV2RoutesMountedAndFunctional(t *testing.T) {
	app := NewApp("127.0.0.1:53217", "")
	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	// Test GET /api/v2/info
	infoReq := httptest.NewRequest(http.MethodGet, "/api/v2/info", nil)
	infoRec := httptest.NewRecorder()
	mux.ServeHTTP(infoRec, infoReq)
	if infoRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v2/info status = %d, want %d", infoRec.Code, http.StatusOK)
	}

	// Test GET /api/v2/transfers
	txReq := httptest.NewRequest(http.MethodGet, "/api/v2/transfers", nil)
	txRec := httptest.NewRecorder()
	mux.ServeHTTP(txRec, txReq)
	if txRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v2/transfers status = %d, want %d", txRec.Code, http.StatusOK)
	}

	// Test GET /api/v2/history
	histReq := httptest.NewRequest(http.MethodGet, "/api/v2/history", nil)
	histRec := httptest.NewRecorder()
	mux.ServeHTTP(histRec, histReq)
	if histRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v2/history status = %d, want %d", histRec.Code, http.StatusOK)
	}
}
