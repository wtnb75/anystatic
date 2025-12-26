package anystatic

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
)

type Config struct {
	RootDir string `json:"rootdir,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

type AnyStatic struct {
	next http.Handler
	hdl  http.Handler
	name string
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.RootDir == "" {
		return nil, fmt.Errorf("rootdir cannot be empty")
	}
	slog.Info("anystatic plugin initialized", "rootdir", config.RootDir)
	fs := os.DirFS(config.RootDir).(fs.StatFS)
	hdl := NewHandler(fs)

	return &AnyStatic{
		next: next,
		hdl:  http.StripPrefix("/", hdl),
		name: name,
	}, nil
}

func (a *AnyStatic) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	a.hdl.ServeHTTP(res, req)
}
