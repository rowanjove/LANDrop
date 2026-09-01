package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func generateQRSVG(content string, size int) ([]byte, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	bitmap := qr.Bitmap()
	n := len(bitmap)
	cellSize := size / n
	if cellSize < 1 {
		cellSize = 1
	}
	totalSize := cellSize * n

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		totalSize, totalSize, totalSize, totalSize)
	svg += fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>`, totalSize, totalSize)

	for y, row := range bitmap {
		for x, cell := range row {
			if cell {
				svg += fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`,
					x*cellSize, y*cellSize, cellSize, cellSize)
			}
		}
	}
	svg += `</svg>`
	return []byte(svg), nil
}

func generateQRASCII(content string) string {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return ""
	}
	return qr.ToSmallString(false)
}

func handleQR(w http.ResponseWriter, r *http.Request, addr string) {
	sizeStr := r.URL.Query().Get("size")
	size := 200
	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 1000 {
			size = s
		}
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target != "" {
		parsed, err := url.Parse(target)
		// QR targets are restricted to an absolute HTTP(S) URL on this server.
		// This prevents the endpoint from becoming a generic phishing/redirect
		// generator while allowing share links to carry their token.
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Host != r.Host {
			target = ""
		}
	}
	if target == "" {
		target = scheme + "://" + addr
	}
	svg, err := generateQRSVG(target, size)
	if err != nil {
		http.Error(w, "failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(svg)
}
