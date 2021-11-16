package lrucache

import (
	"bytes"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"

	lrucache "github.com/9glt/go-caddy-lru-cache/golang-lru"
	simplelru "github.com/9glt/go-caddy-lru-cache/golang-lru/simplelru"
	"golang.org/x/sync/singleflight"
)

var (
	cache simplelru.LRUCache
	sf    singleflight.Group
)

func init() {
	cache, _ = lrucache.NewTTLWithEvict(3000, 60*time.Second, nil)

	caddy.RegisterModule(Middleware{})
	httpcaddyfile.RegisterHandlerDirective("tscache", parseCaddyfile)

	go func() {
		for {
			debug.FreeOSMemory()
			time.Sleep(1 * time.Second)
		}
	}()
}

// Middleware implements an HTTP handler that writes the
// visitor's IP address to a file or stream.
type Middleware struct {
	// The file or stream to write to. Can be "stdout"
	// or "stderr".
	Output string `json:"output,omitempty"`

	w io.Writer
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.tscache",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *Middleware) Provision(ctx caddy.Context) error {
	return nil
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
	if m.w == nil {
		// return fmt.Errorf("no writer")
	}
	return nil
}

type RW struct {
	Bytes *bytes.Buffer
	W     http.ResponseWriter
	Code  int
	H     http.Header
}

func (rw RW) Header() http.Header {
	return rw.H
}

func (rw RW) WriteHeader(status int) {
	rw.Code = status
	rw.W.WriteHeader(status)
}

func (rw RW) Write(b []byte) (int, error) {
	return rw.Bytes.Write(b)
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	var err error
	var value interface{}
	if strings.HasSuffix(r.URL.Path, m.Output) {
		value, err, _ = sf.Do(r.URL.Path, func() (interface{}, error) {
			var value interface{}
			var ok bool
			if value, ok = cache.Get(r.URL.Path); ok {
				return value, nil
			}
			buff := RW{
				Bytes: bytes.NewBuffer(nil),
				W:     w,
				H:     http.Header{},
			}
			err := next.ServeHTTP(buff, r)
			if err == nil {
				cache.Add(r.URL.Path, buff.Bytes.Bytes())
			}
			return buff.Bytes.Bytes(), err
		})
		w.Header().Set("Content-Type", "text/vnd.trolltech.linguist")
		w.Header().Set("Content-Length", len(value.([]byte))
		w.Write(value.([]byte))
		return err
	} else {
		err = next.ServeHTTP(w, r)
	}
	return err
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&m.Output) {
			return d.ArgErr()
		}
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Middleware
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Middleware)(nil)
	_ caddy.Validator             = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
	_ caddyfile.Unmarshaler       = (*Middleware)(nil)
)
