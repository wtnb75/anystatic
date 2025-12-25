package anystatic

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	fs fs.StatFS
}

func NewHandler(fsys fs.StatFS) *Handler {
	return &Handler{fs: fsys}
}

type encodeInfo struct {
	ext    string
	encode string
	order  int
}

var sortorder = map[string]encodeInfo{
	"br":       {ext: ".br", encode: "br", order: 1},
	"zstd":     {ext: ".zst", encode: "zstd", order: 2},
	"gzip":     {ext: ".gz", encode: "gzip", order: 3},
	"deflate":  {ext: ".deflate", encode: "deflate", order: 4},
	"compress": {ext: ".Z", encode: "compress", order: 5},
}

func (h *Handler) accepts(accept string) []encodeInfo {
	enc := []string{}
	for _, v := range strings.Split(accept, ",") {
		vv := strings.SplitN(v, ";", 2)
		enc = append(enc, strings.TrimSpace(vv[0]))
	}
	sort.Slice(enc, func(i, j int) bool {
		vi, iok := sortorder[enc[i]]
		vj, jok := sortorder[enc[j]]
		if iok && jok {
			return vi.order < vj.order
		}
		if iok {
			return true
		}
		return false
	})
	res := []encodeInfo{}
	for _, v := range enc {
		if ei, ok := sortorder[v]; ok {
			res = append(res, ei)
		}
	}
	return res
}

func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var fp fs.File = nil
	path := req.URL.Path
	info, err := h.fs.Stat(path)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		slog.Error("stat failed", "path", path, "error", err)
		return
	}
	content_length := info.Size()
	encoded := false
	for _, ae := range h.accepts(req.Header.Get("Accept-Encoding")) {
		if cinfo, err := h.fs.Stat(path + ae.ext); err == nil {
			if cinfo.ModTime().Round(time.Second).Before(info.ModTime().Round(time.Second)) {
				slog.Warn("encoded file is older than original", "path", path, "ext", ae.ext, "diff", info.ModTime().Sub(cinfo.ModTime()))
				continue
			}
			if cinfo.Size() > info.Size() {
				slog.Info("encoded file is larger than original, skip", "path", path, "ext", ae.ext, "original", info.Size(), "encoded", cinfo.Size())
				continue
			}
			res.Header().Set("Content-Encoding", ae.encode)
			res.Header().Set("Content-Length", strconv.FormatInt(cinfo.Size(), 10))
			ctype := "application/octet-stream"
			if fp0, err := h.fs.Open(path); err == nil {
				defer fp0.Close()
				buf := make([]byte, 512)
				if _, err := fp0.Read(buf); err == nil {
					ctype = http.DetectContentType(buf)
				} else {
					slog.Error("read for content-type failed", "path", path, "error", err)
				}
			} else {
				slog.Error("open original", "path", path, "ext", ae.ext, "error", err)
			}
			res.Header().Set("Content-Type", ctype)
			fp, err = h.fs.Open(path + ae.ext)
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				slog.Error("open error", "path", path, "ext", ae.ext, "error", err)
				return
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
			return
		}
		defer fp.Close()
	}
	if _, err := io.Copy(res, fp); err != nil {
		slog.Error("copy error", "path", path, "error", err)
	}
}
