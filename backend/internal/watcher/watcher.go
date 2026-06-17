package watcher

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

func Watch(path string, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(path); err != nil {
		w.Close()
		return err
	}

	go func() {
		defer w.Close()
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if debounce != nil {
						debounce.Stop()
					}
					debounce = time.AfterFunc(300*time.Millisecond, onChange)
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()
	return nil
}
