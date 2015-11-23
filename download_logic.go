package main

import (
	"fmt"
	"google.golang.org/api/drive/v2"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type getFile struct {
	finished bool

	directory string
	name      string
	id        string
	size      int64
}

// Returns a list of IDs representing the files under the folder given. Returns only one id if it references a file.
func getIdList(cl *drive.Service, basedir string, root string, is_id bool, idchan chan getFile) {
	getIdListRecursive(cl, basedir, root, is_id, idchan)

	// Send one "finished" message per downloader thread
	for i := 0; i < FLAG_par; i++ {
		idchan <- getFile{finished: true}
	}

	return
}

func getIdListRecursive(cl *drive.Service, basedir, root string, is_id bool, idchan chan getFile) {

	if !is_id {
		q := "title = '" + root + "'"

		fl, err := cl.Files.List().Q(q).MaxResults(100).Do()

		if err != nil {
			log.Println(err)
		}

		for i, f := range fl.Items {
			if f.MimeType != "application/vnd.google-apps.folder" {
				if i > 0 {
					idchan <- getFile{directory: basedir, id: f.Id, name: fmt.Sprintf("%d_%s", i, f.Title), size: f.FileSize}
				} else {
					idchan <- getFile{directory: basedir, id: f.Id, name: f.Title, size: f.FileSize}
				}
			} else {
				getIdListRecursive(cl, basedir+f.Title+"/", f.Id, true, idchan)
			}
		}
	} else {
		clist, err := cl.Children.List(root).MaxResults(1000).Do()

		if err != nil {
			log.Println(err)
			return
		}

		if len(clist.Items) == 0 {
			f, err := cl.Files.Get(root).Do()

			if err != nil {
				log.Println(err)
				return
			}
			idchan <- getFile{id: root, directory: basedir, name: f.Title, size: f.FileSize}
		} else {
			for _, child := range clist.Items {
				f, err := cl.Files.Get(child.Id).Do()

				if err != nil {
					log.Println(err)
					return
				}
				if f.MimeType != "application/vnd.google-apps.folder" {
					idchan <- getFile{directory: basedir, id: f.Id, name: f.Title, size: f.FileSize}
				} else {
					getIdListRecursive(cl, basedir+f.Title+"/", f.Id, true, idchan)
				}
			}
		}
	}
}

func getFiles(cl *drive.Service, idchan chan getFile, wg *sync.WaitGroup) error {
	defer wg.Done()

	for file := range idchan {
		if file.finished {
			return nil
		}

		if file.directory != "" {
			os.MkdirAll(file.directory, 0755)
		}

		f, err := os.OpenFile(filepath.Join(file.directory, file.name), os.O_WRONLY|os.O_CREATE, 0644)

		if err != nil {
			log.Println(err)
			continue
		}

		for {
			resp, err := cl.Files.Get(file.id).Download()

			if resp != nil && (resp.StatusCode == 403 || resp.StatusCode >= 500) {
				continue
			}

			if resp != nil && resp.StatusCode == 400 {
				log.Println("Couldn't download", file.name)
				break
			}

			if err != nil {
				log.Println("Couldn't download", file.name)
				log.Println(err)
				break
			}

			var i int64

			fmt.Printf("...%s (%s)\n", file.name, sizeToString(file.size))

			for {
				n, err := io.CopyN(f, resp.Body, file.size/100)
				if err != nil {
					break
				}
				i += n
			}

			fmt.Printf("Finished %s\n", file.name)

			f.Close()
			resp.Body.Close()

			break
		}

	}

	return nil
}
