package web

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog/log"

	"github.com/shutterbase/shutterbase/internal/s3"
)

// Presigner mints short-lived preview URLs for renditions and gallery
// assets. It cannot sign an original: the only object keys it builds are
// "<shard>/<storageId>-<size>.jpg" for a rendition size > 0 and
// "gallery/<key>/…" assets. Originals leave only through the download path.
type Presigner struct {
	client *s3.S3Client
	sizes  []int
	expiry time.Duration
	cache  *expirable.LRU[string, string]
	// base, when set, replaces scheme+host of every signed URL (a dev tunnel
	// fronting the local bucket; the signature stays valid because the tunnel
	// forwards with the real Host).
	base *url.URL
}

func NewPresigner(client *s3.S3Client, sizes []int, expiry, memo time.Duration) *Presigner {
	if memo >= expiry {
		memo = expiry / 2
	}
	return &Presigner{client: client, sizes: sizes, expiry: expiry, cache: expirable.NewLRU[string, string](20000, nil, memo)}
}

// WithBaseURL rewrites signed URLs to scheme://host of base ("" = off).
func (p *Presigner) WithBaseURL(base string) (*Presigner, error) {
	if base == "" {
		return p, nil
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid PRESIGN_BASE_URL %q", base)
	}
	p.base = u
	return p, nil
}

func (p *Presigner) Sizes() []int { return p.sizes }

// Original returns the URL of the full-resolution upload (the hero's zoom
// overlay). Size 0 in GetObjectIds is the unsuffixed key.
func (p *Presigner) Original(ctx context.Context, storageID string) string {
	return p.sign(ctx, s3.GetObjectIds(storageID, nil)[0])
}

// Rendition returns the URL of one preview size; "" when the size is not a
// configured rendition (never falls back to the original).
func (p *Presigner) Rendition(ctx context.Context, storageID string, size int) string {
	if size <= 0 || !p.hasSize(size) {
		return ""
	}
	return p.sign(ctx, s3.GetObjectIds(storageID, []int{size})[size])
}

// Renditions returns every configured size for srcset use.
func (p *Presigner) Renditions(ctx context.Context, storageID string) map[int]string {
	out := make(map[int]string, len(p.sizes))
	for _, size := range p.sizes {
		if u := p.Rendition(ctx, storageID, size); u != "" {
			out[size] = u
		}
	}
	return out
}

// Asset signs a gallery branding object ("gallery/<key>/<id>.<ext>").
func (p *Presigner) Asset(ctx context.Context, key string) string {
	if key == "" || len(key) < len("gallery/") || key[:len("gallery/")] != "gallery/" {
		return ""
	}
	return p.sign(ctx, key)
}

func (p *Presigner) hasSize(size int) bool {
	for _, s := range p.sizes {
		if s == size {
			return true
		}
	}
	return false
}

func (p *Presigner) sign(ctx context.Context, key string) string {
	if p.client == nil {
		return ""
	}
	if u, ok := p.cache.Get(key); ok {
		return u
	}
	u, err := p.client.PresignGet(ctx, key, p.expiry, "")
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("gallery: presign")
		return ""
	}
	if p.base != nil {
		if pu, err := url.Parse(u); err == nil {
			pu.Scheme, pu.Host = p.base.Scheme, p.base.Host
			u = pu.String()
		}
	}
	p.cache.Add(key, u)
	return u
}

// Srcset renders the rendition map as an <img srcset> value.
func Srcset(urls map[int]string) string {
	out := ""
	for _, size := range []int{256, 512, 1024, 2048} {
		if u, ok := urls[size]; ok {
			if out != "" {
				out += ", "
			}
			out += fmt.Sprintf("%s %dw", u, size)
		}
	}
	return out
}
