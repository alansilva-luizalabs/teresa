package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

const (
	PathSeparator = "/"
)

func addAll(tw *tar.Writer, dir string, ignorePatterns []string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "on walk call")
		}

		for _, ip := range ignorePatterns {
			if matched, _ := filepath.Match(ip, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		basePath := fmt.Sprintf("%s%c", filepath.Clean(dir), filepath.Separator)
		name := strings.Replace(path, basePath, "", 1)
		if runtime.GOOS == "windows" {
			path = strings.Replace(path, string(filepath.Separator), PathSeparator, -1)
			name = strings.Replace(name, string(filepath.Separator), PathSeparator, -1)
		}

		return addFile(tw, path, name, info)
	})
}

func addFile(tw *tar.Writer, path, name string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return nil
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return errors.Wrap(err, "failed to build tar header")
	}
	header.Name = name

	if err := tw.WriteHeader(header); err != nil {
		return errors.Wrap(err, "failed to write tar header")
	}

	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return errors.Wrap(err, "failed to copy file contents to tarball")
}

func CreateTemp(dir, prefix string, ignorePatterns []string) (string, error) {
	tmp, err := ioutil.TempFile("", prefix)
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmp.Close()

	gw := gzip.NewWriter(tmp)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := addAll(tw, dir, ignorePatterns); err != nil {
		return "", errors.Wrap(err, "failed to add all files")
	}

	return tmp.Name(), nil
}
