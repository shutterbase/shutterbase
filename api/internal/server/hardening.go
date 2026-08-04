package server

// S10 security hardening of the cookie-auth surface (REWRITE-SPEC §4.13 + §0.10).
// Auth is ambient-cookie now (not bearer), so the mutating surface needs CSRF
// defense; the rest of this file is the rate-limit / body-cap / origin plumbing.
//
// Layout of the two middlewares (order matters):
//   - securityMiddleware: installed BEFORE authentication.Setup so it also wraps
//     the login/auth routes. Does CSRF origin checks, the dev-route gate, the
//     default body cap, and the per-IP login rate limit (no user resolved yet).
//   - rateLimitMiddleware: installed AFTER auth so util.GetUser is populated.
//     Per-user rate limits for /upload-url, POST /images, /download and /ws.
//
// CSRF only fires on mutating methods. GET /download and GET /upload-url are
// cross-origin-read-blocked by CORS (the browser won't expose the response to a
// foreign page), so they are intentionally NOT CSRF-checked here.

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/mxcd/go-config/config"

	"github.com/shutterbase/shutterbase/internal/util"
)

const (
	defaultBodyCap = 1 << 20  // 1 MiB for the small JSON payloads
	imageBodyCap   = 16 << 20 // images carry exifData (PB max was 2 MB); 16 MiB headroom (§0.10)
)

// hardening holds the per-server hardening state built from config in NewServer.
type hardening struct {
	allowedHosts map[string]bool // browser Origin hosts allowed for CSRF / WS upgrade

	loginRL    *rateLimiter
	apiKeyRL   *rateLimiter
	uploadRL   *rateLimiter
	imageRL    *rateLimiter
	downloadRL *rateLimiter
	wsRL       *rateLimiter
}

// buildHardening reads the S10 config knobs. Called once from NewServer.
func buildHardening(apiBaseURL string) *hardening {
	return &hardening{
		allowedHosts: buildAllowedHosts(),
		loginRL:      newRateLimiter(config.Get().Int("RATE_LIMIT_LOGIN_PER_MINUTE")),
		apiKeyRL:     newRateLimiter(config.Get().Int("RATE_LIMIT_APIKEY_PER_MINUTE")),
		uploadRL:     newRateLimiter(config.Get().Int("RATE_LIMIT_UPLOAD_URL_PER_MINUTE")),
		imageRL:      newRateLimiter(config.Get().Int("RATE_LIMIT_IMAGE_CREATE_PER_MINUTE")),
		downloadRL:   newRateLimiter(config.Get().Int("RATE_LIMIT_DOWNLOAD_PER_MINUTE")),
		wsRL:         newRateLimiter(config.Get().Int("RATE_LIMIT_WS_PER_MINUTE")),
	}
}

// buildAllowedHosts collects the browser Origin hosts allowed for mutating
// requests and the WS upgrade: DOMAIN_NAME, the UI_PROXY_URL (DEV Quasar proxy)
// and any CSRF_ALLOWED_ORIGINS entries. Same-origin (Origin host == request
// Host) is always allowed in addition to this set, so the prod reverse-proxy
// host need not be listed explicitly.
func buildAllowedHosts() map[string]bool {
	set := map[string]bool{}
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		// Accept either a full URL (scheme://host[:port]) or a bare host[:port].
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			set[strings.ToLower(u.Host)] = true
			return
		}
		set[strings.ToLower(raw)] = true
	}
	add(config.Get().String("DOMAIN_NAME"))
	add(config.Get().String("UI_PROXY_URL"))
	for _, o := range strings.Split(config.Get().String("CSRF_ALLOWED_ORIGINS"), ",") {
		add(o)
	}
	return set
}

// originAllowed returns true when the browser Origin host matches the request
// Host (same-origin) or is in the configured allow-list.
func (h *hardening) originAllowed(reqHost, originHost string) bool {
	if originHost == "" {
		return false
	}
	if strings.EqualFold(originHost, reqHost) {
		return true
	}
	return h.allowedHosts[strings.ToLower(originHost)]
}

// csrfOK enforces a strict same-Origin/Referer check on mutating requests. Safe
// methods pass through. A mutating request MUST carry an Origin (or Referer)
// whose host is allowed, else it is rejected — a forged cross-site POST from a
// foreign page either omits Origin or carries the attacker's, both rejected.
func (h *hardening) csrfOK(c *gin.Context) bool {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = c.GetHeader("Referer")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return h.originAllowed(c.Request.Host, u.Host)
}

// wsOriginOK gates the WS upgrade. Unlike csrfOK it allows a missing Origin so
// non-browser clients (the time-sync tooling) can still connect; a present
// Origin must be same-origin or allow-listed.
func (h *hardening) wsOriginOK(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return h.originAllowed(r.Host, u.Host)
}

// securityMiddleware runs before auth: dev-route gate, CSRF, default body cap,
// and the per-IP login rate limit.
func (s *Server) securityMiddleware(apiBaseURL string) gin.HandlerFunc {
	devPrefix := apiBaseURL + "/dev/"
	loginPath := apiBaseURL + "/auth/login"
	return func(c *gin.Context) {
		// Dev-route gate: /api/v1/dev/* 404s when DEV=false (dev routes land in
		// Wave 4; assert the gate now so they can never leak into prod).
		if !s.options.DevMode && strings.HasPrefix(c.Request.URL.Path, devPrefix) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if !s.hardening.csrfOK(c) {
			apiError(c, http.StatusForbidden, "csrf_origin", "origin not allowed")
			return
		}

		// Default body cap; image create/update get the larger cap (set in the
		// images controller). Only caps requests that carry a body.
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !isImageWriteRoute(c) {
				c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, defaultBodyCap)
			}
		}

		if c.Request.Method == http.MethodPost && c.Request.URL.Path == loginPath {
			if !s.hardening.loginRL.allow("ip:" + c.ClientIP()) {
				tooMany(c)
				return
			}
		}

		// S-review #7: API-key requests are authenticated by the api-key middleware
		// (downstream) and never hit the per-user limiter on a bad key, so a flood of
		// invalid keys would otherwise hammer the argon2 verifier unbounded. Cap it
		// pre-auth, per IP, here — before the key is ever looked up / verified.
		if strings.HasPrefix(c.GetHeader("Authorization"), "ApiKey ") {
			if !s.hardening.apiKeyRL.allow("apikey-ip:" + c.ClientIP()) {
				tooMany(c)
				return
			}
		}
		c.Next()
	}
}

// rateLimitMiddleware runs after auth (util.GetUser populated): per-user limits
// on the upload/image/download/ws surface.
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var rl *rateLimiter
		switch c.FullPath() {
		case s.options.ApiBaseURL + "/upload-url":
			rl = s.hardening.uploadRL
		case s.options.ApiBaseURL + "/download/:id/:res":
			rl = s.hardening.downloadRL
		case "/ws":
			rl = s.hardening.wsRL
		case s.options.ApiBaseURL + "/images":
			if c.Request.Method == http.MethodPost {
				rl = s.hardening.imageRL
			}
		}
		if rl != nil && !rl.allow(userOrIPKey(c)) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

func isImageWriteRoute(c *gin.Context) bool {
	p := c.FullPath()
	return strings.HasSuffix(p, "/images") && c.Request.Method == http.MethodPost ||
		strings.HasSuffix(p, "/images/:id") && c.Request.Method == http.MethodPut
}

func userOrIPKey(c *gin.Context) string {
	if u := util.GetUser(c.Request.Context()); u != nil {
		return "u:" + u.ID.String()
	}
	return "ip:" + c.ClientIP()
}

func tooMany(c *gin.Context) {
	apiError(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
}

// parseTrustedProxies splits the comma-separated TRUSTED_PROXIES config into the
// []string gin.SetTrustedProxies wants. Empty/blank entries are dropped; an empty
// result returns nil so gin trusts NO proxy (ClientIP == real RemoteAddr).
func parseTrustedProxies(raw string) []string {
	out := []string{}
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// rateLimiter is a per-key token bucket. Keys are evicted by an LRU so memory is
// bounded; a re-created bucket starts full, which only ever loosens the limit.
// ponytail: in-memory, per-instance; swap for a shared store only if multi-replica.
type rateLimiter struct {
	ratePerSec float64
	burst      float64
	buckets    *expirable.LRU[string, *tokenBucket]
}

type tokenBucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

// newRateLimiter builds a limiter of perMinute requests/min with a burst equal to
// the per-minute budget. perMinute <= 0 disables limiting (allow-all).
func newRateLimiter(perMinute int) *rateLimiter {
	return &rateLimiter{
		ratePerSec: float64(perMinute) / 60.0,
		burst:      float64(perMinute),
		buckets:    expirable.NewLRU[string, *tokenBucket](8192, nil, 10*time.Minute),
	}
}

func (rl *rateLimiter) allow(key string) bool {
	if rl.burst <= 0 {
		return true
	}
	// ponytail: Get/Add isn't atomic, so a racing pair can briefly mint two
	// buckets for the same key — harmless, it only loses one token of accounting.
	b, ok := rl.buckets.Get(key)
	if !ok {
		b = &tokenBucket{tokens: rl.burst, last: time.Now()}
		rl.buckets.Add(key, b)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * rl.ratePerSec
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
