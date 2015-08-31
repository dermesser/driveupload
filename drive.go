package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var FLAG_folder_id string

// May chdir() into another directory; returns name of directory to start uploading
func getStartDir() string {
	path := flag.Arg(0)
	dir, file := filepath.Split(path)
	filename := ""

	if dir == "" && file != "" {
		// e.g. abc
		filename = file
	} else if dir != "" && file == "" {
		// e.g. abc/xyz/
		// chdir into directory and do it from there
		filename = "."
		if err := os.Chdir(dir); err != nil {
			log.Fatal(err)
		}
	} else {
		// e.g. abc/xyz
		// chdir into directory and do it from there
		filename = file
		if err := os.Chdir(dir); err != nil {
			log.Fatal(err)
		}
	}

	return filename
}

func getFileList(local_folder string) []string {
	list := make([]string, 0, 16)

	st, err := os.Stat(local_folder)

	if err != nil {
		log.Fatal(err)
		return list
	}

	if !st.IsDir() {
		return append(list, local_folder)
	}

	files, err := ioutil.ReadDir(local_folder)

	if err != nil {
		log.Fatal(err)
		return list
	}

	for _, f := range files {
		if f.IsDir() {
			list = append(list, getFileList(filepath.Join(local_folder, f.Name()))...)
		} else {
			list = append(list, filepath.Join(local_folder, f.Name()))
		}
	}

	return list
}
func registerFlags() {
	flag.StringVar(&FLAG_folder_id, "folder", "root", "A folder ID (can be taken from the Drive Web URL) of the folder to put files in")
}

func main() {

	registerFlags()

	flag.Parse()

	driveclient, err := getDriveClient()

	if err != nil {
		log.Fatal(err)
	}

	var filelist []string
	filename := getStartDir()

	filelist = getFileList(filename)

	err = uploadFileList(driveclient, filelist)

	if err != nil {
		log.Fatal(err)
	}
}
