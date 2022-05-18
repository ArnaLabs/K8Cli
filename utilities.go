package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// referred from https://golangcode.com/download-a-file-from-a-url/

type StrSlice []string

func (list StrSlice) Has(a string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func getFileFromURL(fileName string, fileUrl string) {
	err := DownloadFile(fileName, fileUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println("Downloaded: " + fileUrl)

}

func makeDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}

func deleteDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}

	}
	os.Remove(dir)
	return nil
}
