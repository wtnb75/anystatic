package anystatic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// TestAccepts_EmptyAcceptEncoding tests accepts() with empty header
func TestAccepts_EmptyAcceptEncoding(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("")

	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

// TestAccepts_SingleEncoding tests accepts() with single encoding
func TestAccepts_SingleEncoding(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("gzip")

	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
		return
	}

	if result[0].encode != "gzip" {
		t.Errorf("expected 'gzip', got %q", result[0].encode)
	}
}

// TestAccepts_MultipleEncodings tests accepts() with multiple encodings in priority order
func TestAccepts_MultipleEncodings(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("gzip, deflate, br")

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
		return
	}

	// Expected order: br (order:1), gzip (order:3), deflate (order:4)
	expected := []string{"br", "gzip", "deflate"}
	for i, exp := range expected {
		if result[i].encode != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, result[i].encode)
		}
	}
}

// TestAccepts_WithQualityValues tests that accepts() ignores quality values
func TestAccepts_WithQualityValues(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("gzip;q=1.0, deflate;q=0.5")

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
		return
	}

	// Quality values are ignored; priority order is used instead
	// gzip (order:3), deflate (order:4)
	expected := []string{"gzip", "deflate"}
	for i, exp := range expected {
		if result[i].encode != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, result[i].encode)
		}
	}
}

// TestAccepts_UnknownEncoding tests that unknown encodings are filtered out
func TestAccepts_UnknownEncoding(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("unknown, gzip, br")

	if len(result) != 2 {
		t.Errorf("expected 2 items (unknown filtered), got %d", len(result))
		return
	}

	expected := []string{"br", "gzip"}
	for i, exp := range expected {
		if result[i].encode != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, result[i].encode)
		}
	}
}

// TestAccepts_WithSpaces tests accepts() handles extra whitespace
func TestAccepts_WithSpaces(t *testing.T) {
	h := NewHandler(fstest.MapFS{})
	result := h.accepts("  gzip  ,  deflate  ")

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
		return
	}

	expected := []string{"gzip", "deflate"}
	for i, exp := range expected {
		if result[i].encode != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, result[i].encode)
		}
	}
}

// TestServeHTTP_FileNotFound tests 404 response
func TestServeHTTP_FileNotFound(t *testing.T) {
	fsys := fstest.MapFS{}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/missing.txt", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestServeHTTP_BasicFileServing tests basic file serving with correct headers
func TestServeHTTP_BasicFileServing(t *testing.T) {
	fsys := fstest.MapFS{
		"test.txt": &fstest.MapFile{Data: []byte("Hello, World!")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "Hello, World!" {
		t.Errorf("expected body 'Hello, World!', got %q", body)
	}

	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Errorf("expected Content-Type header, got empty")
	}

	if cl := w.Header().Get("Content-Length"); cl != "13" {
		t.Errorf("expected Content-Length: 13, got %q", cl)
	}

	if vary := w.Header().Get("Vary"); vary != "Accept-Encoding" {
		t.Errorf("expected Vary: Accept-Encoding, got %q", vary)
	}
}

// TestServeHTTP_DirectoryIndexHTML tests automatic index.html serving
func TestServeHTTP_DirectoryIndexHTML(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>Home</html>")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "<html>Home</html>" {
		t.Errorf("expected body '<html>Home</html>', got %q", body)
	}
}

// TestServeHTTP_DirectoryIndexHTMLWithTrailingSlash tests /path/ -> /path/index.html
func TestServeHTTP_DirectoryIndexHTMLWithTrailingSlash(t *testing.T) {
	fsys := fstest.MapFS{
		"dir/index.html": &fstest.MapFile{Data: []byte("<html>Dir</html>")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/dir/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "<html>Dir</html>" {
		t.Errorf("expected body '<html>Dir</html>', got %q", body)
	}
}

// TestServeHTTP_CompressedFilePreferred tests compressed file is served when available
func TestServeHTTP_CompressedFilePreferred(t *testing.T) {
	fsys := fstest.MapFS{
		"test.txt":    &fstest.MapFile{Data: []byte("original content here")},
		"test.txt.gz": &fstest.MapFile{Data: []byte("compressed gz content")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "compressed gz content" {
		t.Errorf("expected compressed content, got %q", body)
	}

	if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %q", enc)
	}
}

// TestServeHTTP_NoMatchingCompression tests fallback when client doesn't accept compression
func TestServeHTTP_NoMatchingCompression(t *testing.T) {
	fsys := fstest.MapFS{
		"test.txt":    &fstest.MapFile{Data: []byte("original content")},
		"test.txt.gz": &fstest.MapFile{Data: []byte("compressed")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	// No Accept-Encoding header
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "original content" {
		t.Errorf("expected original content, got %q", body)
	}

	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("expected no Content-Encoding, got %q", enc)
	}
}

// TestServeHTTP_CompressedFileLargerThanOriginal tests skipping larger compressed files
func TestServeHTTP_CompressedFileLargerThanOriginal(t *testing.T) {
	fsys := fstest.MapFS{
		"small.txt":    &fstest.MapFile{Data: []byte("x")},
		"small.txt.gz": &fstest.MapFile{Data: []byte("this is a very long compressed file that is larger than the original")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/small.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "x" {
		t.Errorf("expected original content 'x', got %q", body)
	}

	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("expected no Content-Encoding (fallback), got %q", enc)
	}
}

// TestServeHTTP_EncodingPriority tests br is preferred over gzip
func TestServeHTTP_EncodingPriority(t *testing.T) {
	fsys := fstest.MapFS{
		"test.txt":    &fstest.MapFile{Data: []byte("original content that is long enough to compress well")},
		"test.txt.br": &fstest.MapFile{Data: []byte("br")},
		"test.txt.gz": &fstest.MapFile{Data: []byte("gz")},
	}
	h := NewHandler(fsys)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if body := w.Body.String(); body != "br" {
		t.Errorf("expected br compressed content, got %q", body)
	}

	if enc := w.Header().Get("Content-Encoding"); enc != "br" {
		t.Errorf("expected Content-Encoding: br, got %q", enc)
	}
}

// TestServeHTTP_AllEncodingPriorities tests full priority order: br > zstd > gzip > deflate > compress
func TestServeHTTP_AllEncodingPriorities(t *testing.T) {
	testCases := []struct {
		name       string
		header     string
		available  map[string]string
		expectEnc  string
		expectBody string
	}{
		{
			name:       "br preferred over gzip",
			header:     "gzip, br",
			available:  map[string]string{"": "orig", ".br": "br", ".gz": "gz"},
			expectEnc:  "br",
			expectBody: "br",
		},
		{
			name:       "zstd preferred over gzip",
			header:     "gzip, zstd",
			available:  map[string]string{"": "orig", ".zst": "zst", ".gz": "gz"},
			expectEnc:  "zstd",
			expectBody: "zst",
		},
		{
			name:       "br preferred over all",
			header:     "gzip, zstd, br, deflate",
			available:  map[string]string{"": "orig", ".br": "br", ".zst": "zst", ".gz": "gz", ".deflate": "def"},
			expectEnc:  "br",
			expectBody: "br",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsysMap := make(fstest.MapFS)
			for ext, content := range tc.available {
				fsysMap["test.txt"+ext] = &fstest.MapFile{Data: []byte(content)}
			}

			h := NewHandler(fsysMap)
			req := httptest.NewRequest("GET", "/test.txt", nil)
			req.Header.Set("Accept-Encoding", tc.header)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			if enc := w.Header().Get("Content-Encoding"); enc != tc.expectEnc {
				t.Errorf("expected Content-Encoding: %s, got %q", tc.expectEnc, enc)
			}

			if body := w.Body.String(); body != tc.expectBody {
				t.Errorf("expected body %q, got %q", tc.expectBody, body)
			}
		})
	}
}
