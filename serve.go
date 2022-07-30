package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/danprince/sietch/internal/errors"
	"github.com/fsnotify/fsnotify"
)

var buildErr error

func Serve(b *builder) {
	server := http.FileServer(http.Dir(b.outDir))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if buildErr != nil {
			w.Header().Add("Content-type", "text/html")
			w.WriteHeader(500)
			w.Write([]byte(errors.FmtErrorHtml(buildErr)))
		} else {
			server.ServeHTTP(w, r)
		}
	}))

	go watch(func(events []fsnotify.Event) []string {
		var dt time.Duration
		var watchDirs []string
		dt, buildErr = b.build()
		defer b.reset()

		if buildErr != nil {
			fmt.Fprintln(os.Stderr, errors.FmtError(buildErr))
		} else {
			fmt.Printf("built %d pages in %s\n", len(b.pages), dt)
			watchDirs = b.dirs[:]
		}

		// Watch all crawled directories for changes.
		for _, dir := range b.dirs {
			watchDirs = append(watchDirs, path.Join(b.pagesDir, dir))
		}

		return watchDirs
	})

	fmt.Println("serving site at http://localhost:8000...")
	err := http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func watch(build func(events []fsnotify.Event) []string) {
	watcher, err := fsnotify.NewWatcher()
	ticker := time.Tick(time.Millisecond * 100)

	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	done := make(chan bool)

	var events []fsnotify.Event

	// Trigger an initial build with no changes
	watchDirs := build(events)

	for _, dir := range watchDirs {
		watcher.Add(dir)
	}

	for {
		select {
		case <-done:
			return

		case event := <-watcher.Events:
			if event.Op != fsnotify.Chmod {
				events = append(events, event)
			}

		case err := <-watcher.Errors:
			log.Println(errors.FmtError(err))

		case <-ticker:
			if len(events) > 0 {
				dirs := build(events)
				events = []fsnotify.Event{}

				for _, dir := range watcher.WatchList() {
					watcher.Remove(dir)
				}

				for _, dir := range dirs {
					watcher.Add(dir)
				}
			}
		}
	}
}
