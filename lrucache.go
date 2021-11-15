package lrucache

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Middleware{})
	httpcaddyfile.RegisterHandlerDirective("visitor_ip", parseCaddyfile)
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
		ID:  "http.handlers.visitor_ip",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *Middleware) Provision(ctx caddy.Context) error {
	switch m.Output {
	case "stdout":
		m.w = os.Stdout
	case "stderr":
		m.w = os.Stderr
	default:
		return fmt.Errorf("an output stream is required")
	}
	return nil
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
	if m.w == nil {
		return fmt.Errorf("no writer")
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
	// fmt.Printf("%s\n", b)
	rw.Bytes.Write(b)
	return rw.W.Write(b)
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// m.w.Write([]byte(r.RemoteAddr))
	// fmt.Printf("%v\n", r)
	buff := RW{
		Bytes: bytes.NewBuffer(nil),
		W:     w,
		H:     http.Header{},
	}
	err := next.ServeHTTP(buff, r)
	fmt.Printf("%s", buff.Bytes.Bytes())
	fmt.Printf("%v", buff.H)
	// w.WriteHeader(buff.Code)
	// w.Header().Add("Content-Length", fmt.Sprintf("%d", buff.Bytes.Len()))
	// w.Write(buff.Bytes.Bytes())
	// buff.Bytes.WriteTo(w)
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
