package main

import (
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watches dir recursively, ignoring directories that match patterns in excludes
// which are checked with filepath.Match. Events are batched and sent in groups
// at most every 100ms.
func watch(dir string, excludes []string) chan []fsnotify.Event {
	w, err := fsnotify.NewWatcher()
	ch := make(chan []fsnotify.Event)

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer w.Close()
		ticker := time.Tick(100 * time.Millisecond)
		queue := []fsnotify.Event{}

		for {
			for _, dir := range w.WatchList() {
				w.Remove(dir)
			}

			filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					for _, pat := range excludes {
						if match, _ := filepath.Match(pat, path); match {
							return filepath.SkipDir
						}
					}
					w.Add(path)
				}
				return nil
			})

		polling:
			for {
				select {
				case err := <-w.Errors:
					log.Fatal(err)
				case e := <-w.Events:
					if e.Op != fsnotify.Chmod {
						queue = append(queue, e)
					}
				case <-ticker:
					if len(queue) > 0 {
						ch <- queue
						queue = []fsnotify.Event{}
						break polling
					}
				}
			}
		}
	}()

	return ch
}
