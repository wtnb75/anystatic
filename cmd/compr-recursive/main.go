package main

import (
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/buildkite/shellwords"
)

// compress:
//   - compress all files in tree, keep original, parallel
//   - check timestamp
//   - gzip -k9nf, brotli -k9nf, zstd -k19f
//   - remove if larger than original
//
// cleanup:
//   - remove all compressed files in tree

type compressor struct {
	ext string
	cmd []string
}

var compressors = []compressor{
	{ext: ".gz", cmd: []string{"gzip", "-k9nf"}},
	{ext: ".br", cmd: []string{"brotli", "-k9nf"}},
	{ext: ".zst", cmd: []string{"zstd", "-k19f"}},
}

func compressFile(root fs.FS, basepath, path string, dry bool) error {
	stfs := root.(fs.StatFS)
	origst, err := stfs.Stat(path)
	if err != nil {
		slog.Error("stat failed in compressFile", "path", path, "error", err)
		return err
	}
	for _, v := range compressors {
		outfn := path + v.ext
		absfn := filepath.Join(basepath, path)
		st, err := stfs.Stat(outfn)
		if err == nil {
			// check timestamp
			if st.ModTime().After(origst.ModTime()) {
				slog.Info("skip compressing, up-to-date", "path", path, "compressed", outfn)
				continue
			}
		}
		cmd := v.cmd[:]
		cmd = append(cmd, absfn)
		if dry {
			slog.Info("dry-run: would compress file", "path", path, "cmd", cmd)
		} else {
			if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
				slog.Error("compress failed", "path", path, "cmd", cmd)
				return err
			}
			st, err = stfs.Stat(outfn)
			if err != nil {
				slog.Error("stat compressed file failed", "path", outfn, "error", err)
				return err
			}
			if st.Size() >= origst.Size() {
				slog.Info("compressed file is larger than original, removing", "path", path, "compressed", outfn, "original_size", origst.Size(), "compressed_size", st.Size())
				if err := os.Remove(filepath.Join(".", outfn)); err != nil {
					slog.Error("remove compressed file failed", "path", outfn, "error", err)
					return err
				}
			} else {
				slog.Info("compressed file created", "path", path, "compressed", outfn, "original_size", origst.Size(), "compressed_size", st.Size())
			}
		}
	}
	return nil
}

func cleanupFile(root fs.FS, basepath, path string, cleanold bool, dry bool) error {
	stfs := root.(fs.StatFS)
	origst, err := stfs.Stat(path)
	if err != nil {
		slog.Error("stat failed", "path", path, "error", err)
		return err
	}
	for _, v := range compressors {
		outfn := path + v.ext
		absfn := filepath.Join(basepath, outfn)
		st, err := stfs.Stat(outfn)
		if err != nil {
			slog.Debug("not exists?", "path", outfn)
			continue
		}
		if cleanold {
			if st.ModTime().After(origst.ModTime()) {
				slog.Info("skip cleanup, up-to-date", "path", path, "compressed", outfn)
				continue
			}
		}
		if dry {
			slog.Info("dry-run: would cleanup file", "path", path, "compressed", outfn)
		} else {
			if err := os.Remove(absfn); err != nil {
				slog.Error("remove compressed file failed", "path", outfn, "error", err)
				return err
			}
			slog.Info("removed compressed file", "path", path, "compressed", outfn)
		}
	}
	return nil
}

func main() {
	var dry bool
	var rootfs fs.FS

	cmpr := flag.NewFlagSet("compress", flag.ExitOnError)
	cmprdir := cmpr.String("dir", "", "target directory")
	cmprdry := cmpr.Bool("dry-run", false, "dry run")
	minsize := cmpr.Int64("min-size", 128, "minimum file size to compress")
	maxsize := cmpr.Int64("max-size", 10*1024*1024, "maximum file size to compress")
	gzipcmd := cmpr.String("gzip-cmd", "", "gzip command")
	brotlicmd := cmpr.String("brotli-cmd", "", "brotli command")
	zstdcmd := cmpr.String("zstd-cmd", "", "zstd command")
	cleanup := flag.NewFlagSet("cleanup", flag.ExitOnError)
	cleanold := cleanup.Bool("old", false, "remove only old compressed files")
	cleandir := cleanup.String("dir", "", "target directory")
	cleandry := cleanup.Bool("dry-run", false, "dry run")

	args := os.Args[1:]
	slog.Info("args", "args", args)

	if len(args) == 0 {
		slog.Error("subcommand is required")
		panic("subcommand is required")
	}

	switch args[0] {
	case "compress":
		if err := cmpr.Parse(args[1:]); err != nil {
			slog.Error("parse error", "error", err)
			panic(err)
		}
		if *cmprdir == "" {
			slog.Error("dir is required")
			panic("dir is required")
		}
		rootfs = os.DirFS(*cmprdir)
		if *gzipcmd != "" {
			compressors[0].cmd, _ = shellwords.Split(*gzipcmd)
		}
		if *brotlicmd != "" {
			compressors[1].cmd, _ = shellwords.Split(*brotlicmd)
		}
		if *zstdcmd != "" {
			compressors[2].cmd, _ = shellwords.Split(*zstdcmd)
		}
		dry = *cmprdry
	case "cleanup":
		if err := cleanup.Parse(args[1:]); err != nil {
			slog.Error("parse error", "error", err)
			panic(err)
		}
		if *cleandir == "" {
			slog.Error("dir is required")
			panic("dir is required")
		}
		rootfs = os.DirFS(*cleandir)
		dry = *cleandry
	default:
		slog.Error("unknown subcommand", "subcommand", args)
		panic("unknown subcommand: " + args[0])
	}

	err := fs.WalkDir(rootfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		abspath := path
		switch filepath.Ext(d.Name()) {
		case ".gz", ".br", ".zst", ".deflate", ".Z":
			return nil
		default:
			if cmpr.Parsed() {
				st, err := d.Info()
				if err != nil {
					slog.Error("stat failed", "path", abspath, "error", err)
					panic(err)
				}
				if st.Size() < *minsize {
					slog.Info("skip compressing, too small", "path", abspath, "size", st.Size(), "min_size", *minsize)
					return nil
				}
				if st.Size() > *maxsize {
					slog.Info("skip compressing, too large", "path", abspath, "size", st.Size(), "max_size", *maxsize)
					return nil
				}
				slog.Info("compressing file", "path", abspath)
				if err := compressFile(rootfs, *cmprdir, abspath, dry); err != nil {
					slog.Error("compress failed", "path", abspath, "error", err)
					panic(err)
				}
			} else if cleanup.Parsed() {
				slog.Info("cleanup file", "path", abspath, "old_only", *cleanold)
				if err := cleanupFile(rootfs, *cleandir, abspath, *cleanold, dry); err != nil {
					slog.Error("cleanup failed", "path", abspath, "error", err)
					panic(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("walk failed", "dir", rootfs, "error", err)
		panic(err)
	}
}
