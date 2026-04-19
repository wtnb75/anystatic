package anystatic

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	pathpkg "path"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	fs               fs.StatFS
	logAccessHeaders bool
}

type HandlerOption func(*Handler)

func WithAccessLogHeaders(enabled bool) HandlerOption {
	return func(h *Handler) {
		h.logAccessHeaders = enabled
	}
}

func NewHandler(fsys fs.StatFS, opts ...HandlerOption) *Handler {
	slog.Info("handler created", "root", fsys)
	h := &Handler{fs: fsys, logAccessHeaders: true}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

type encodeInfo struct {
	ext    string
	encode string
	order  int
}

var sortorder = map[string]encodeInfo{
	// brotli vs zstd: which is winner?
	"br":       {ext: ".br", encode: "br", order: 1},
	"zstd":     {ext: ".zst", encode: "zstd", order: 2},
	"gzip":     {ext: ".gz", encode: "gzip", order: 3},
	"deflate":  {ext: ".deflate", encode: "deflate", order: 4},
	"compress": {ext: ".Z", encode: "compress", order: 5},
}

var contentTypesByExt = map[string]string{
	".css":  "text/css; charset=utf-8",
	".gif":  "image/gif",
	".htm":  "text/html; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".ico":  "image/x-icon",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".js":   "text/javascript; charset=utf-8",
	".json": "application/json",
	".map":  "application/json",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".txt":  "text/plain; charset=utf-8",
	".wasm": "application/wasm",
	".webp": "image/webp",
	".xml":  "text/xml; charset=utf-8",
}

func (h *Handler) accepts(accept string) []encodeInfo {
	if accept == "" {
		return nil
	}

	counts := map[string]int{}
	known := 0
	for _, v := range strings.Split(accept, ",") {
		vv := strings.SplitN(v, ";", 2)
		enc := strings.TrimSpace(vv[0])
		if _, ok := sortorder[enc]; ok {
			counts[enc]++
			known++
		}
	}
	if known == 0 {
		return nil
	}

	res := make([]encodeInfo, 0, known)
	for _, key := range []string{"br", "zstd", "gzip", "deflate", "compress"} {
		ei := sortorder[key]
		for i := 0; i < counts[key]; i++ {
			res = append(res, ei)
		}
	}
	return res
}

func (h *Handler) serveHTTP(res http.ResponseWriter, req *http.Request) int {
	var fp fs.File = nil
	path := strings.TrimPrefix(req.URL.Path, "/")
	if path == "" || strings.HasSuffix(path, "/") {
		path += "index.html"
	}
	info, err := h.fs.Stat(path)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		slog.Error("stat failed", "path", path, "error", err)
		return http.StatusNotFound
	}
	content_length := info.Size()
	infoModSec := info.ModTime().Round(time.Second)
	encoded := false
	ctype := contentTypesByExt[pathpkg.Ext(path)]
	if ctype == "" {
		ctype = "application/octet-stream"
		if fp0, err := h.fs.Open(path); err == nil {
			defer fp0.Close()
			buf := make([]byte, 512)
			if n, err := fp0.Read(buf); err == nil || err == io.EOF {
				if n > 0 {
					ctype = http.DetectContentType(buf[:n])
				}
			} else {
				slog.Error("read for content-type failed", "path", path, "error", err)
			}
		} else {
			slog.Error("open original", "path", path, "error", err)
		}
	}
	res.Header().Set("Content-Type", ctype)
	res.Header().Set("Vary", "Accept-Encoding")
	for _, ae := range h.accepts(req.Header.Get("Accept-Encoding")) {
		encodedPath := path + ae.ext
		if cinfo, err := h.fs.Stat(encodedPath); err == nil {
			if cinfo.ModTime().Round(time.Second).Before(infoModSec) {
				slog.Warn("encoded file is older than original", "path", path, "ext", ae.ext, "diff", info.ModTime().Sub(cinfo.ModTime()))
				continue
			}
			if cinfo.Size() > info.Size() {
				slog.Info("encoded file is larger than original, skip", "path", path, "ext", ae.ext, "original", info.Size(), "encoded", cinfo.Size())
				continue
			}
			res.Header().Set("Content-Encoding", ae.encode)
			res.Header().Set("Content-Length", strconv.FormatInt(cinfo.Size(), 10))
			fp, err = h.fs.Open(encodedPath)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				slog.Error("open error", "path", path, "ext", ae.ext, "error", err)
				return http.StatusInternalServerError
			}
			defer fp.Close()
			slog.Debug("encoded file", "path", path, "ext", ae.ext)
			encoded = true
			break
		}
	}
	if !encoded {
		res.Header().Set("Content-Length", strconv.FormatInt(content_length, 10))
		fp, err = h.fs.Open(path)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			slog.Error("open error", "path", path, "error", err)
			return http.StatusInternalServerError
		}
		defer fp.Close()
	}
	if _, err := io.Copy(res, fp); err != nil {
		slog.Error("copy error", "path", path, "error", err)
	}
	return http.StatusOK
}

func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	st := time.Now()
	code := h.serveHTTP(res, req)
	attrs := []any{"method", req.Method, "path", req.URL.Path, "remote", req.RemoteAddr, "status", code, "elapsed_ns", time.Since(st)}
	if h.logAccessHeaders {
		attrs = append(attrs, "req-header", req.Header, "res-header", res.Header())
	}
	slog.Info("accesslog", attrs...)
}
