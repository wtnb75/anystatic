package main

import (
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/wtnb75/anystatic"
)

func realMain() error {
	listen := flag.String("listen", ":8800", "listen address")
	dir := flag.String("dir", ".", "serve directory")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()
	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	slog.SetLogLoggerLevel(level)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	fs := os.DirFS(*dir).(fs.StatFS)
	hdl := anystatic.NewHandler(fs)
	server := http.Server{
		Addr:    *listen,
		Handler: http.StripPrefix("/", hdl),
	}
	slog.Info("starting server", "addr", server.Addr)
	return server.ListenAndServe()
}

func main() {
	if err := realMain(); err != nil {
		slog.Error("server error", "error", err)
	}
}
