package main

import (
	"net/http"
	"sync"
	"time"
)

type PINManager struct {
	mu       sync.Mutex
	pin      string
	attempts map[string]*attemptInfo
}

type attemptInfo struct {
	count    int
	lockedAt time.Time
}

const (
	maxAttempts = 5
	lockDuration = 60 * time.Second
)

func NewPINManager(pin string) *PINManager {
	return &PINManager{
		pin:      pin,
		attempts: make(map[string]*attemptInfo),
	}
}

func (p *PINManager) IsEnabled() bool {
	return p.pin != ""
}

func (p *PINManager) Verify(ip string, inputPIN string) (bool, int, bool) {
	if !p.IsEnabled() {
		return true, 0, false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	info, exists := p.attempts[ip]
	if exists && info.count >= maxAttempts {
		if time.Since(info.lockedAt) < lockDuration {
			return false, 0, true // locked
		}
		// Reset after lock duration
		info.count = 0
	}

	if inputPIN == p.pin {
		if exists {
			delete(p.attempts, ip)
		}
		return true, 0, false
	}

	// Wrong PIN
	if !exists {
		info = &attemptInfo{}
		p.attempts[ip] = info
	}
	info.count++
	remaining := maxAttempts - info.count
	if info.count >= maxAttempts {
		info.lockedAt = time.Now()
		return false, 0, true
	}
	return false, remaining, false
}

func (p *PINManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Allow info and qr endpoints without PIN
		if r.URL.Path == "/info" || r.URL.Path == "/qr" {
			next.ServeHTTP(w, r)
			return
		}

		// Check session cookie
		cookie, err := r.Cookie("landrop_pin")
		if err == nil && cookie.Value == p.pin {
			next.ServeHTTP(w, r)
			return
		}

		// Check PIN header
		pin := r.Header.Get("X-LanDrop-PIN")
		if pin == "" {
			pin = r.URL.Query().Get("pin")
		}

		if pin == "" {
			// Return PIN entry page for browser requests
			if r.Header.Get("Accept") == "" || r.URL.Path == "/" {
				servePINPage(w)
				return
			}
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error": "PIN required",
				"code":  "PIN_REQUIRED",
			})
			return
		}

		ip := r.RemoteAddr
		ok, remaining, locked := p.Verify(ip, pin)
		if locked {
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"error": "已锁定，请 60 秒后重试",
				"code":  "PIN_LOCKED",
			})
			return
		}
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error":     "PIN 码错误",
				"code":      "PIN_WRONG",
				"remaining": remaining,
			})
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "landrop_pin",
			Value: p.pin,
			Path:  "/",
		})

		next.ServeHTTP(w, r)
	})
}

func servePINPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>LAN Drop - PIN</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui;display:flex;justify-content:center;align-items:center;min-height:100vh;background:#f5f5f5}
@media(prefers-color-scheme:dark){body{background:#1a1a1a;color:#eee}}
.card{background:#fff;border-radius:12px;padding:2rem;box-shadow:0 2px 12px rgba(0,0,0,.1);text-align:center;max-width:320px;width:90%}
@media(prefers-color-scheme:dark){.card{background:#2a2a2a}}
h2{margin-bottom:1rem;font-size:1.2rem}
input{width:100%;padding:.75rem;font-size:1.5rem;text-align:center;border:2px solid #ddd;border-radius:8px;letter-spacing:.5em;margin-bottom:1rem}
@media(prefers-color-scheme:dark){input{background:#333;border-color:#555;color:#eee}}
button{width:100%;padding:.75rem;background:#4A90D9;color:#fff;border:none;border-radius:8px;font-size:1rem;cursor:pointer}
button:hover{background:#357ABD}
.error{color:#e74c3c;margin-top:.5rem;font-size:.9rem;display:none}
</style></head><body>
<div class="card">
<h2>🔒 PIN Required</h2>
<form id="f"><input id="p" type="password" maxlength="4" pattern="\d{4}" placeholder="····" autofocus>
<button type="submit">Unlock</button></form>
<div class="error" id="e"></div>
</div>
<script>
document.getElementById('f').onsubmit=async e=>{
e.preventDefault();const p=document.getElementById('p').value;
const r=await fetch('/info',{headers:{'X-LanDrop-PIN':p}});
if(r.ok){document.cookie='landrop_pin='+p+';path=/';location.reload()}
else{const d=await r.json();const el=document.getElementById('e');
el.textContent=d.error||'PIN incorrect';el.style.display='block'}};
</script></body></html>`))
}
