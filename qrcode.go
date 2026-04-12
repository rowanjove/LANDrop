package main

import (
	"fmt"
	"net/http"
	"strconv"

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

	url := "http://" + addr
	svg, err := generateQRSVG(url, size)
	if err != nil {
		http.Error(w, "failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(svg)
}
