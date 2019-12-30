package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type infile struct {
	name      string
	basenames map[string]*zip.File
	reader    *zip.ReadCloser
}

func newInfile(fname string) (*infile, error) {
	reader, err := zip.OpenReader(fname)
	if err != nil {
		return nil, fmt.Errorf("cannot open archive: %v", err)
	}
	names := make(map[string]*zip.File)
	for _, f := range reader.File {
		if f.Mode().IsDir() {
			continue
		}
		names[path.Base(f.Name)] = f
	}
	return &infile{
		name:      fname,
		reader:    reader,
		basenames: names,
	}, nil
}

func (f *infile) pick(basename string, fn func(r io.Reader) error) error {
	fl, ok := f.basenames[basename]
	if !ok {
		return fmt.Errorf("file %s not in zipfile %s", basename, f.name)
	}
	rc, err := fl.Open()
	if err != nil {
		return fmt.Errorf("cannot open %s inside %s: %v", basename, f.name, err)
	}
	defer rc.Close()
	return fn(rc)
}

func (f *infile) rezip(fname string, basenames []string) error {
	for _, basename := range basenames {
		if _, ok := f.basenames[basename]; !ok {
			return fmt.Errorf("required file %s is not in %s", basename, f.name)
		}
	}
	// TODO atomic temp file write
	zf, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("cannot open destination zipfile: %v", err)
	}
	w := zip.NewWriter(zf)
	for _, basename := range basenames {
		in, err := f.basenames[basename].Open()
		if err != nil {
			return fmt.Errorf("cannot open %s inside %s: %v", basename, f.name, err)
		}
		out, err := w.Create(basename)
		if err != nil {
			return fmt.Errorf("cannot get source file from zip: %v", err)
		}
		if _, err := io.Copy(out, in); err != nil {
			return fmt.Errorf("cannot copy file into zip archive: %v", err)
		}
		in.Close()
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("cannot close zip writer: %v", err)
	}
	if err := zf.Close(); err != nil {
		return fmt.Errorf("cannot close destination zipfile: %v", err)
	}
	return nil
}

func (f *infile) close() {
	f.reader.Close()
}

func main() {
	infile, err := newInfile("testdata/test.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer infile.close()
	infile.pick("test1", func(r io.Reader) error {
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			return fmt.Errorf("cannot read file contents: %v", err)
		}
		return nil
	})
	infile.pick("test2", func(r io.Reader) error {
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			return fmt.Errorf("cannot read file contents: %v", err)
		}
		return nil
	})
	infile.pick("test3", func(r io.Reader) error {
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			return fmt.Errorf("cannot read file contents: %v", err)
		}
		return nil
	})
	if err := infile.rezip("testdata/out.zip", []string{"test1", "test2"}); err != nil {
		log.Fatal(err)
	}
}
