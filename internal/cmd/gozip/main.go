package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/johnstarich/go/datasize"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Path to Go source is required")
		os.Exit(1)
		return
	}
	err := archiveGo(os.Args[1], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func archiveGo(goRoot string, w io.Writer) error {
	compressor, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return err
	}
	archive := tar.NewWriter(compressor)

	goRoot, err = filepath.Abs(goRoot)
	if err != nil {
		return err
	}

	doFile := func(path string, info os.FileInfo) error {
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(path, goRoot)
		err = archive.WriteHeader(header)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(archive, f)
		return err
	}

	goBinary := filepath.Join(goRoot, "bin", "js_wasm", "go")
	goBinaryInfo, err := os.Stat(goBinary)
	if err != nil {
		return err
	}
	err = doFile(goBinary, goBinaryInfo)
	if err != nil {
		return err
	}

	stats, err := walkGo(goRoot, doFile)
	fmt.Fprintf(os.Stderr, "Stats: %s\n", stats)
	if err != nil {
		return err
	}

	err = archive.Close()
	if err != nil {
		return err
	}
	return compressor.Close()
}

type Int64Slice []int64

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Stats struct {
	Visited      int
	SkippedDirs  int
	IgnoredFiles int

	sizes []int64
}

func (s Stats) SizeMetrics() (mean, median, max float64) {
	sort.Sort(Int64Slice(s.sizes))
	var total float64
	for _, num := range s.sizes {
		total += float64(num)
		if n := float64(num); n > max {
			max = n
		}
	}
	return total / float64(len(s.sizes)), float64(s.sizes[len(s.sizes)/2]), max
}

func (s Stats) String() string {
	mean, median, max := s.SizeMetrics()
	size := func(i int64) string {
		return datasize.Bytes(int64(i)).String()
	}
	return fmt.Sprintf("mean=%v, median=%v, max=%v, 90th%%=%v, 99th%%=%v, visited=%v, skipped dirs=%v ignored files=%v", size(int64(mean)), size(int64(median)), size(int64(max)), size(s.sizes[len(s.sizes)*90/100]), size(s.sizes[len(s.sizes)*99/100]), s.Visited, s.SkippedDirs, s.IgnoredFiles)
}

// walkGo walks through a Go sources directory root and runs 'do' on files to archive.
func walkGo(goRoot string, do func(string, os.FileInfo) error) (Stats, error) {
	var stats Stats
	walkPath := goRoot + string(filepath.Separator) // ensures symlink dir is followed
	return stats, filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		stats.sizes = append(stats.sizes, info.Size())

		switch {
		case path == ".":
			return nil // "skip" top-level dir, don't record in stats
		case matchPath(path, goRoot, ".git"),
			matchPath(path, goRoot, "api"),
			matchPath(path, goRoot, "doc"),
			matchPath(path, goRoot, "src", "cmd"),
			matchPath(path, goRoot, "src", "runtime", "cgo"),
			matchPath(path, goRoot, "src", "runtime", "race"),
			strings.HasSuffix(path, string(filepath.Separator)+"testdata"),
			matchPath(path, goRoot, "test"):
			stats.SkippedDirs++
			return filepath.SkipDir // explicitly skip all of these contents
		case matchPath(path, goRoot, "pkg", "tool", "js_wasm", "cgo"),
			matchPath(path, goRoot, "bin", "js_wasm", "go"), // handled specially above
			strings.HasSuffix(path, ".a"),
			strings.HasSuffix(path, "_test.go"):
			return nil // skip specific files
		case matchPathPrefix(path, goRoot, "bin", "js_wasm"),
			matchPathPrefix(path, goRoot, "pkg", "js_wasm"),
			matchPathPrefix(path, goRoot, "pkg", "include"),
			matchPathPrefix(path, goRoot, "pkg", "tool", "js_wasm"):
			stats.Visited++
			return do(path, info)
		case matchPathPrefix(path, goRoot, "bin"),
			matchPathPrefix(path, goRoot, "pkg"):
			stats.IgnoredFiles++
			return nil // skip things not explicitly matched here
		default:
			stats.Visited++
			return do(path, info)
		}
	})
}

func matchPath(match string, paths ...string) bool {
	return match == filepath.Join(paths...)
}

// matchPathPrefix returns true if joining paths forms a prefix for or is equal to 'match'
func matchPathPrefix(match string, paths ...string) bool {
	path := filepath.Join(paths...)
	return match == path || strings.HasPrefix(match, path+string(filepath.Separator))
}
