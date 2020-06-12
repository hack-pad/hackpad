package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func main() {
	fmt.Println("go-wasm")
	err := run3()
	if err != nil {
		fmt.Println("Error", err)
	}
}

func run3() error {
	err := ioutil.WriteFile("main.go", []byte(`
package main

func main() {
	println("hello world")
}
`), 0750)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("go.mod", []byte(`
module thing
`), 0750)
}

func run2() error {
	fmt.Println("run2")
	if err := os.Mkdir("/go", 0750); err != nil {
		return errors.Wrap(err, "Failed to make dir /go")
	}

	err := ioutil.WriteFile("/go/test.txt", []byte("hello world"), 0750)
	if err != nil {
		return errors.Wrap(err, "Failed to write /go/test.txt")
	}

	dir, err := ioutil.ReadDir("/go")
	if err != nil {
		return errors.Wrap(err, "Failed to read dir /go")
	}
	fmt.Println("contents:")
	for _, f := range dir {
		fmt.Println(f.Name())
	}
	info, err := os.Stat("/go")
	if err != nil {
		return err
	}
	fmt.Println("is dir: ", info.IsDir())
	fmt.Println("dir perms:", info.Mode())
	return nil
}

func run() error {
	resp, err := http.Get("/go.zip")
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := bytes.NewReader(buf)

	err = unzip(r, r.Len(), "/go")
	fmt.Println(err)

	dir, err := ioutil.ReadDir("/go")
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, f := range dir {
		fmt.Println(f.Name())
	}
	info, err := os.Stat("/go")
	if err != nil {
		return err
	}
	fmt.Println("dir perm", info.Mode())
	return nil
}

func unzip(r io.ReaderAt, size int, outPath string) error {
	if err := os.Mkdir(outPath, 0750); err != nil {
		return err
	}

	return nil
	z, err := zip.NewReader(r, int64(size))
	if err != nil {
		return err
	}
	return unzipFiles(z.File, outPath)
}

func unzipFiles(files []*zip.File, path string) error {
	for _, f := range files {
		fPath := filepath.Join(path, f.Name)
		unzipFile(f, fPath)
	}
	return nil
}

func unzipFile(file *zip.File, dest string) error {
	if file.FileInfo().IsDir() {
		return os.Mkdir(dest, file.Mode())
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, buf, file.Mode())
}
