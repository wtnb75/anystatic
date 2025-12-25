package main

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/wtnb75/anystatic"
)

func realMain() error {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	fs := os.DirFS(".").(fs.StatFS)
	hdl := anystatic.NewHandler(fs)
	server := http.Server{
		Addr:    ":8800",
		Handler: http.StripPrefix("/", hdl),
	}
	return server.ListenAndServe()
}

func main() {
	if err := realMain(); err != nil {
		slog.Error("server error", "error", err)
	}
}
