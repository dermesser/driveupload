package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/context"
	"google.golang.org/api/drive/v2"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var clearline string = fmt.Sprintf("%c[2K\r", 27)
var current_file string

func sizeToString(size int64) string {
	prefix := ""
	flsz := float64(0)

	if size > 1<<30 {
		prefix = "G"
		flsz = float64(size) / float64(1<<30)
	} else if size > 1<<20 {
		prefix = "M"
		flsz = float64(size) / float64(1<<20)
	} else if size > 1<<10 {
		prefix = "k"
		flsz = float64(size) / float64(1<<10)
	} else {
		flsz = float64(size)
	}

	return fmt.Sprintf("%.1f%s", flsz, prefix)
}

func getProgressFunction(filename string) func(done, max int64) {
	return func(done, max int64) {
		if current_file == "" {
			current_file = filename
		}
		if filename != current_file {
			current_file = filename
			fmt.Println()
		}

		width, _, _ := terminal.GetSize(0)
		bar_width := width - 33

		n_filled := int((float64(done) / float64(max)) * float64(bar_width))

		sizestr := sizeToString(done)

		if n_filled > len(sizestr) {
			n_filled -= len(sizestr)
		} else {
			n_filled = 0
		}

		n_empty := bar_width - n_filled - len(sizestr)

		filled := strings.Repeat("=", n_filled) + sizestr
		empty := strings.Repeat(" ", n_empty)

		short_filename := filename

		if len(filename) > 30 {
			short_filename = "..." + filename[len(filename)-27:len(filename)]
		}

		bar := fmt.Sprintf("%30s [%s%s]", short_filename, filled, empty)

		fmt.Print(clearline)
		fmt.Print(bar)
	}
}

func uploadFileList(cl *drive.Service, list []string) error {
	pathtoidmap := make(map[string]string)
	pathtoidmap["."] = FLAG_folder_id

	for _, file := range list {
		dir := filepath.Dir(file)

		id, ok := pathtoidmap[dir]

		if !ok {
			err := createNestedFolders(cl, pathtoidmap, FLAG_folder_id, strings.Split(dir, "/"))
			if err != nil {
				return err
			}
			id = pathtoidmap[dir]
		}

		parentref := &drive.ParentReference{Id: id}
		drive_file := &drive.File{Title: filepath.Base(file), Parents: []*drive.ParentReference{parentref}}

		data, err := os.Open(file)

		if err != nil {
			log.Print(err.Error())
			continue
		}

		stat, err := data.Stat()

		_, err = cl.Files.Insert(drive_file).
			ResumableMedia(context.Background(), data, stat.Size(), "").
			ProgressUpdater(getProgressFunction(file)).
			Do()

		data.Close()

		if err != nil {
			log.Print(err.Error())
		}
	}
	fmt.Println("")

	return nil
}

func createNestedFolders(cl *drive.Service, pathtoid map[string]string, root string, folders []string) error {
	parent := root
	path := ""

	for _, folder := range folders {
		if folder == "" || folder == "." {
			continue
		}

		path = filepath.Join(path, folder)

		next_parent, ok := pathtoid[path]

		if ok {
			parent = next_parent
			continue
		} else {

			query := "title = '" + folder + "' and '" + parent + "' in parents"

			list, err := cl.Files.List().Q(query).Do()

			if err != nil {
				return err
			}

			if len(list.Items) == 0 {
				parentref := &drive.ParentReference{Id: parent}
				file := &drive.File{MimeType: "application/vnd.google-apps.folder",
					Title: folder, Parents: []*drive.ParentReference{parentref}}

				f, err := cl.Files.Insert(file).Do()

				if err != nil {
					return err
				}
				pathtoid[path] = f.Id
				parent = f.Id
			} else {
				pathtoid[path] = list.Items[0].Id
				parent = list.Items[0].Id
			}
		}
	}

	return nil
}
