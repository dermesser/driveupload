package main

import (
	"flag"
	"log"
)

var FLAG_folder_id string
var FLAG_recursive bool

func registerFlags() {
	flag.StringVar(&FLAG_folder_id, "folder", "root", "A folder ID (can be taken from the Drive Web URL) of the folder to put files in")
	flag.BoolVar(&FLAG_recursive, "r", false, "Upload recursively")
}

func main() {

	registerFlags()

	flag.Parse()

	driveclient, err := getDriveClient()

	if err != nil {
		log.Fatal(err)
	}

	filename := flag.Arg(0)
	var filelist []string

	if FLAG_recursive {
		filelist = getFileList(filename)
	} else {
		filelist = []string{filename}
	}

	err = uploadFileList(driveclient, filelist)

	if err != nil {
		log.Fatal(err)
	}
}
