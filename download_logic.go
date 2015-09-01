package main

import (
	"google.golang.org/api/drive/v2"
	"io"
	"log"
	"os"
)

type getFile struct {
	directory string
	name      string
	id        string
	size      int64
}

// Returns a list of IDs representing the files under the folder given. Returns only one id if it references a file.
func getIdList(cl *drive.Service, basedir string, root string, is_id bool) []getFile {
	idlist := make([]getFile, 0, 8)

	if !is_id {
		q := "title = '" + root + "'"

		fl, err := cl.Files.List().Q(q).MaxResults(100).Do()

		if err != nil {
			log.Println(err)
		}

		for _, f := range fl.Items {
			if f.MimeType != "application/vnd.google-apps.folder" {
				idlist = append(idlist, getFile{directory: basedir, id: f.Id, name: f.Title, size: f.FileSize})
			} else {
				idlist = append(idlist, getIdList(cl, basedir+f.Title+"/", f.Id, true)...)
			}
		}
	} else {
		clist, err := cl.Children.List(root).MaxResults(100).Do()

		if err != nil {
			log.Println(err)
		}

		if len(clist.Items) == 0 {
			f, err := cl.Files.Get(root).Do()

			if err != nil {
				log.Println(err)
				return nil
			}
			idlist = append(idlist, getFile{id: root, directory: basedir, name: f.Title, size: f.FileSize})
		} else {
			for _, child := range clist.Items {
				f, err := cl.Files.Get(child.Id).Do()

				if err != nil {
					log.Println(err)
					continue
				}
				if f.MimeType != "application/vnd.google-apps.folder" {
					idlist = append(idlist, getFile{directory: basedir, id: f.Id, name: f.Title, size: f.FileSize})
				} else {
					idlist = append(idlist, getIdList(cl, basedir+f.Title+"/", f.Id, true)...)
				}
			}
		}
	}

	return idlist
}

func getFiles(cl *drive.Service, idlist []getFile) error {
	olddir, _ := os.Getwd()

	for _, file := range idlist {
		if file.directory != "" {
			os.MkdirAll(file.directory, 0755)
			os.Chdir(file.directory)
		}

		f, err := os.OpenFile(file.name, os.O_WRONLY|os.O_CREATE, 0644)

		if err != nil {
			log.Println(err)
			continue
		}
		resp, err := cl.Files.Get(file.id).Download()

		if err != nil {
			log.Println(err)
			continue
		}

		var i int64
		progress_func := getProgressFunction(file.name)

		for {
			n, err := io.CopyN(f, resp.Body, file.size/100)
			if err != nil {
				break
			}
			i += n
			progress_func(i, file.size)
		}

		f.Close()
		resp.Body.Close()

		os.Chdir(olddir)
	}

	return nil
}
