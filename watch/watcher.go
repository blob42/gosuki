package watch

import (
	"git.sp4ke.xyz/sp4ke/gomark/logging"

	"github.com/fsnotify/fsnotify"
)

var log = logging.GetLogger("WATCH")

type WatchRunner interface {
	Watcher
	Runner
}

// If the browser needs the watcher to be reset for each new event
type ResetWatcher interface {
	ResetWatcher() // resets a new watcher
}

// Required interface to be implemented by browsers that want to use the
// fsnotify event loop and watch changes on bookmark files.
type Watcher interface {
	Watcher() *WatchDescriptor
}

type Runner interface {
	Run()
}

// Wrapper around fsnotify watcher
type WatchDescriptor struct {
	ID      string
	W       *fsnotify.Watcher // underlying fsnotify watcher
	Watched map[string]*Watch // watched paths
	Watches []*Watch          // helper var

	// channel used to communicate watched events
	eventsChan chan fsnotify.Event
    isWatching bool
}

func (w WatchDescriptor) hasReducer() bool {
	//TODO: test the type of eventsChan
	return w.eventsChan != nil
}

func NewWatcherWithReducer(name string, reducerLen int, watches ...*Watch) (*WatchDescriptor, error) {
	w, err := NewWatcher(name, watches...)
	if err != nil {
		return nil, err
	}
	w.eventsChan = make(chan fsnotify.Event, reducerLen)

	return w, nil
}

func NewWatcher(name string, watches ...*Watch) (*WatchDescriptor, error) {

	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watchedMap := make(map[string]*Watch)
	for _, v := range watches {
		watchedMap[v.Path] = v
	}

	watcher := &WatchDescriptor{
		ID:         name,
		W:          fswatcher,
		Watched:    watchedMap,
		Watches:    watches,
		eventsChan: nil,
	}

	// Add all watched paths
	for _, v := range watches {

		err = watcher.W.Add(v.Path)
		if err != nil {
			return nil, err
		}
	}
	return watcher, nil
}

// Details about the object being watched
type Watch struct {
	Path       string        // Path to watch for events
	EventTypes []fsnotify.Op // events to watch for
	EventNames []string      // event names to watch for (file/dir names)

	// Reset the watcher at each event occurence (useful for create events)
	ResetWatch bool
}

func SpawnWatcher(w WatchRunner) {
    watcher := w.Watcher()
    if ! watcher.isWatching {
        go WatcherThread(w)
        watcher.isWatching = true

		for watched := range watcher.Watched{
			log.Infof("Watching %s", watched)
		}
    }

}

// Main thread for watching file changes
func WatcherThread(w WatchRunner) {

	watcher := w.Watcher()
	log.Infof("<%s> Started watcher", watcher.ID)
	for {
		// Keep watcher here as it is reset from within
		// the select block
		resetWatch := false

		select {
		case event := <-watcher.W.Events:
			// Very verbose
			//log.Debugf("event: %v | eventName: %v", event.Op, event.Name)

			// On Chrome like browsers the bookmarks file is created
			// at every change.

			/*
			 * When a file inside a watched directory is renamed/created,
			 * fsnotify does not seem to resume watching the newly created file, we
			 * need to destroy and create a new watcher. The ResetWatcher() and
			 * `break` statement ensure we get out of the `select` block and catch
			 * the newly created watcher to catch events even after rename/create
			 */

			for _, watched := range watcher.Watches {
				for _, watchedEv := range watched.EventTypes {
					for _, watchedName := range watched.EventNames {
						if event.Op&watchedEv == watchedEv &&
							event.Name == watchedName {

							// For watchers who need a reducer
							// to avoid spammy events
							if watcher.hasReducer() {
								ch := watcher.eventsChan
								ch <- event
							} else {
								w.Run()
							}

							//log.Warningf("event: %v | eventName: %v", event.Op, event.Name)

							if watched.ResetWatch {
								log.Debugf("resetting watchers")
								if r, ok := w.(ResetWatcher); ok {
									r.ResetWatcher()
									resetWatch = true // needed to break out of big loop
								} else {
									log.Fatalf("<%s> does not implement ResetWatcher", watcher.ID)
								}
							}

						}
					}
				}
			}

			if resetWatch {
				break
			}

			// Firefox keeps the file open and makes changes on it
			// It needs a debouncer
			//if event.Name == bookmarkPath {
			//log.Debugf("event: %v | eventName: %v", event.Op, event.Name)
			////go debounce(1000*time.Millisecond, spammyEventsChannel, w)
			//ch := w.EventsChan()
			//ch <- event
			////w.Run()
			//}
		case err := <-watcher.W.Errors:
			if err != nil {
				log.Error(err)
			}
		}
	}
}