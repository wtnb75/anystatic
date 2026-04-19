package anystatic

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func init() {
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func benchmarkPayload(size int) []byte {
	if size <= 0 {
		return nil
	}
	base := []byte("anystatic benchmark payload line\n")
	n := size/len(base) + 1
	data := bytes.Repeat(base, n)
	return data[:size]
}

func BenchmarkAccepts(b *testing.B) {
	h := NewHandler(fstest.MapFS{})
	header := "gzip, deflate, br, zstd, compress"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := h.accepts(header)
		if len(res) == 0 {
			b.Fatal("accepts returned empty")
		}
	}
}

func BenchmarkServeHTTP_NoCompression(b *testing.B) {
	data := benchmarkPayload(256 * 1024)
	fsys := fstest.MapFS{
		"large.txt": &fstest.MapFile{Data: data},
	}
	h := NewHandler(fsys)
	req := httptest.NewRequest(http.MethodGet, "/large.txt", nil)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

func BenchmarkServeHTTP_GzipPreferred(b *testing.B) {
	original := benchmarkPayload(256 * 1024)
	compressed := benchmarkPayload(32 * 1024)
	fsys := fstest.MapFS{
		"large.txt":    &fstest.MapFile{Data: original},
		"large.txt.gz": &fstest.MapFile{Data: compressed},
	}
	h := NewHandler(fsys)
	req := httptest.NewRequest(http.MethodGet, "/large.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ReportAllocs()
	b.SetBytes(int64(len(compressed)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
		if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
			b.Fatalf("unexpected content-encoding: %q", enc)
		}
	}
}

func BenchmarkServeHTTP_Parallel(b *testing.B) {
	data := benchmarkPayload(32 * 1024)
	fsys := fstest.MapFS{
		"parallel.txt": &fstest.MapFile{Data: data},
	}
	h := NewHandler(fsys)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/parallel.txt", nil)
		for pb.Next() {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", w.Code)
			}
		}
	})
}
