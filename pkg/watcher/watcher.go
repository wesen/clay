package watcher

import (
	"context"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
)

type Callback func(path string) error

// Watcher provides a way to recursively watch a set of paths for changes.
// It recursively adds all directories present (and created in the future)
// to provide coverage.
//
// You can provide a doublestar mask to filter out paths. For example, to
// only watch for changes to .txt files, you can provide "**/*.txt".
type Watcher struct {
	paths    []string
	masks    []string
	callback Callback
}

// Run is a blocking loop that will watch the paths provided and call the
func (w *Watcher) Run(ctx context.Context) error {
	if w.callback == nil {
		return errors.New("no callback provided")
	}

	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Add each path to the watcher
	for _, path := range w.paths {
		log.Debug().Str("path", path).Msg("Adding recursive path to watcher")
		err = addRecursive(watcher, path)
		if err != nil {
			return err
		}
	}

	log.Info().Strs("paths", w.paths).Strs("masks", w.masks).Msg("Watching paths")

	// Listen for events until the context is cancelled
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			log.Debug().Str("event", event.String()).Msg("Received fsnotify event")

			// if it is a deletion, remove the directory from the watcher
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Debug().Str("path", event.Name).Msg("Removing directory from watcher")
				err = removePathsWithPrefix(watcher, event.Name)
				if err != nil {
					return err
				}
				continue
			}

			// if a new directory is created, add it to the watcher
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err != nil {
					return err
				}
				if info.IsDir() {
					log.Debug().Str("path", event.Name).Msg("Adding new directory to watcher")
					err = addRecursive(watcher, event.Name)
					if err != nil {
						return err
					}
					continue
				}
			}

			if len(w.masks) > 0 {
				matched := false
				for _, mask := range w.masks {
					doesMatch, err := doublestar.Match(mask, event.Name)
					if err != nil {
						return err
					}

					if doesMatch {
						matched = true
						break
					}

				}

				if !matched {
					log.Debug().Str("path", event.Name).Strs("masks", w.masks).Msg("Skipping event because it does not match the mask")
					continue
				}
			}

			if event.Op&fsnotify.Write != fsnotify.Write && event.Op&fsnotify.Create != fsnotify.Create {
				log.Debug().Str("path", event.Name).Msg("Skipping event because it is not a write or create event")
				continue
			}
			log.Info().Str("path", event.Name).Msg("File modified")
			if w.callback != nil {
				err = w.callback(event.Name)
				if err != nil {
					return err
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(err).Msg("Received fsnotify error")
		}
	}
}

type Option func(w *Watcher)

func WithPaths(paths ...string) Option {
	return func(w *Watcher) {
		w.paths = append(w.paths, paths...)
	}
}

func WithMask(masks ...string) Option {
	return func(w *Watcher) {
		w.masks = masks
	}
}

func NewWatcher(callback Callback, options ...Option) *Watcher {
	ret := &Watcher{
		paths:    []string{},
		callback: callback,
		masks:    []string{},
	}

	for _, opt := range options {
		opt(ret)
	}

	return ret
}

// removePathsWithPrefix removes `name` and all subdirectories from the watcher
func removePathsWithPrefix(watcher *fsnotify.Watcher, name string) error {
	// we do the "recursion" by checking the watchlist of the watcher for all watched directories
	// that has name as prefix
	watchlist := watcher.WatchList()
	for _, path := range watchlist {
		if strings.HasPrefix(path, name) {
			log.Debug().Str("path", path).Msg("Removing path from watcher")
			err := watcher.Remove(path)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Recursively add a path to the watcher
func addRecursive(watcher *fsnotify.Watcher, path string) error {
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	addPath := strings.TrimSuffix(path, string(os.PathSeparator))

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// check if we have permissions to watch
	if info.Mode()&os.ModeSymlink != 0 {
		log.Debug().Str("path", addPath).Msg("Skipping symlink")
		return nil
	}

	// open and then close to check if we can actually read from the file
	f, err := os.Open(addPath)
	if err != nil {
		log.Warn().Str("path", addPath).Msg("Skipping path because we cannot read it")
		return nil
	}
	_ = f.Close()

	log.Debug().Str("path", addPath).Msg("Adding path to watcher")
	err = watcher.Add(addPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		log.Debug().Str("path", path).Msg("Walking path to add subpaths to watcher")
		err = filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {
			if err != nil {
				log.Warn().Err(err).Str("path", subpath).Msg("Error walking path")
				return nil
			}
			if subpath == path {
				return nil
			}
			log.Trace().Str("path", subpath).Msg("Testing subpath to watcher")
			if info.IsDir() {
				log.Debug().Str("path", subpath).Msg("Adding subpath to watcher")
				err = addRecursive(watcher, subpath)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
