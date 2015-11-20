package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var FLAG_folder_id string
var FLAG_get bool
var FLAG_par int

var DOWNLOAD_FINISHED bool

// May chdir() into another directory; returns name of directory to start uploading
func getStartDir(path string) string {
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
	flag.BoolVar(&FLAG_get, "get", false, "Download files. Either the folder given by -folder or the file or folder given as name")
	flag.IntVar(&FLAG_par, "par", 4, "How many files to download in parallel")
}

func main() {

	registerFlags()

	flag.Parse()

	driveclient, err := getDriveClient()

	if err != nil {
		log.Fatal(err)
	}

	if !FLAG_get {
		orig_dir, err := os.Getwd()

		for _, path := range flag.Args() {
			var filelist []string

			filename := getStartDir(path)

			filelist = getFileList(filename)

			err = uploadFileList(driveclient, filelist)

			if err != nil {
				log.Fatal(err)
			}

			os.Chdir(orig_dir)
		}
	} else {
		// getIdList sends file metadata here, getFiles receives IDs and starts downloading
		fileid_chan := make(chan getFile, 256)

		fmt.Println("Getting file list...")

		if FLAG_folder_id != "root" {
			go getIdList(driveclient, "", FLAG_folder_id, true, fileid_chan)
		} else {
			go getIdList(driveclient, "", flag.Arg(0), false, fileid_chan)
		}

		fmt.Println("Downloading items...")

		wg := new(sync.WaitGroup)

		for i := 0; i < FLAG_par; i++ {
			wg.Add(1)
			go getFiles(driveclient, fileid_chan, wg)
		}
		wg.Wait()
	}
}
