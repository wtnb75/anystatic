package main

import (
	"flag"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/wtnb75/anystatic"
)

func do_listen(listen string) (net.Listener, error) {
	protos := strings.SplitN(listen, ":", 2)
	switch protos[0] {
	case "unix", "tcp", "tcp4", "tcp6":
		return net.Listen(protos[0], protos[1])
	}
	return net.Listen("tcp", listen)
}

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
		Handler: hdl,
	}
	listener, err := do_listen(*listen)
	if err != nil {
		slog.Error("listen error", "error", err)
		return err
	}
	defer listener.Close()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		slog.Info("shutting down server")
		listener.Close()
	}()
	slog.Info("starting server", "addr", server.Addr)
	return server.Serve(listener)
}

func main() {
	if err := realMain(); err != nil {
		slog.Error("server error", "error", err)
	}
}
