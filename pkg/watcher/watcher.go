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

type WriteCallback func(path string) error
type RemoveCallback func(path string) error

// Watcher provides a way to recursively watch a set of paths for changes.
// It recursively adds all directories present (and created in the future)
// to provide coverage.
//
// You can provide a doublestar mask to filter out paths. For example, to
// only watch for changes to .txt files, you can provide "**/*.txt".
type Watcher struct {
	paths          []string
	masks          []string
	writeCallback  WriteCallback
	removeCallback RemoveCallback
	breakOnError   bool
}

// Run is a blocking loop that will watch the paths provided and call the
func (w *Watcher) Run(ctx context.Context) error {
	if w.writeCallback == nil {
		return errors.New("no writeCallback provided")
	}

	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func(watcher *fsnotify.Watcher) {
		_ = watcher.Close()
	}(watcher)

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
			log.Debug().Msg("Context cancelled, stopping watcher")
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			log.Debug().Str("op", event.Op.String()).
				Str("name", event.Name).
				Strs("watchList", watcher.WatchList()).
				Msg("Received fsnotify event")

			// if it is a deletion, remove the directory from the watcher
			// TODO(manuel, 2023-03-27) There's a race condition here where a rename is a Create followed by a Remove.
			// See also https://github.com/go-go-golems/cliopatra/issues/10
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				err = removePathsWithPrefix(watcher, event.Name)
				if err != nil {
					log.Warn().Err(err).Str("path", event.Name).Msg("Could not remove path from watcher")
					if w.breakOnError {
						return err
					}
				}
			}

			if event.Op&fsnotify.Rename == fsnotify.Rename {
				err = removePathsWithPrefix(watcher, event.Name)
				if err != nil {
					log.Warn().Err(err).Str("path", event.Name).Msg("Could not remove path from watcher")
					if w.breakOnError {
						return err
					}
				}
			}

			// if a new directory is created, add it to the watcher
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)

				if err != nil {
					log.Debug().Err(err).Str("path", event.Name).Msg("Could not stat path")
					continue
				}

				// if a directory was created, we always add it. For files, we must first check if it matches
				// the globs.
				if info.IsDir() {
					log.Debug().Str("path", event.Name).Msg("Adding new directory to watcher")
					err = addRecursive(watcher, event.Name)
					if err != nil {
						log.Warn().Err(err).Str("path", event.Name).Msg("Could not add directory to watcher")
						if w.breakOnError {
							return err
						}
					}
					continue
				}
			}

			// check if the path matches the globs
			if len(w.masks) > 0 {
				matched := false
				for _, mask := range w.masks {
					doesMatch, err := doublestar.Match(mask, event.Name)
					if err != nil {
						log.Warn().Err(err).Str("path", event.Name).Str("mask", mask).Msg("Could not match path with mask")
						if w.breakOnError {
							return err
						}
						break
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

			if event.Name == "" {
				// This is a workaround for a bug in fsnotify where the name is empty.
				// This might be a race condition with removing the renamed file and adding it again.
				// The Rename event is triggered by MOVE_FROM. Ignoring the empty path seems to work fine,
				// the Rename on the proper path is triggered afterwards.
				continue
			}

			// if the new file is valid, add it to the watcher for changes and removal
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Debug().Str("path", event.Name).Msg("Adding path to watchlist")
				err = watcher.Add(event.Name)
				if err != nil {
					log.Warn().Err(err).Str("path", event.Name).Msg("Could not add path to watcher")
					if w.breakOnError {
						return err
					}
				}
			}

			isWriteEvent := event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create
			isRemoveEvent := event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Remove == fsnotify.Remove

			if isWriteEvent && w.writeCallback != nil {
				err = w.writeCallback(event.Name)
				if err != nil {
					log.Warn().Err(err).Str("path", event.Name).Msg("Error while processing write event")
					if w.breakOnError {
						return err
					}
				}
			}

			if isRemoveEvent && w.removeCallback != nil {
				err = w.removeCallback(event.Name)
				if err != nil {
					log.Warn().Err(err).Str("path", event.Name).Msg("Error while processing remove event")
					if w.breakOnError {
						return err
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(err).Msg("Received fsnotify error")
			if w.breakOnError {
				return err
			}
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

func WithWriteCallback(callback WriteCallback) Option {
	return func(w *Watcher) {
		w.writeCallback = callback
	}
}

func WithRemoveCallback(callback RemoveCallback) Option {
	return func(w *Watcher) {
		w.removeCallback = callback
	}
}

func WithBreakOnError(breakOnError bool) Option {
	return func(w *Watcher) {
		w.breakOnError = breakOnError
	}
}

func NewWatcher(options ...Option) *Watcher {
	ret := &Watcher{
		paths: []string{},
		masks: []string{},
	}

	for _, opt := range options {
		opt(ret)
	}

	return ret
}

// removePathsWithPrefix removes `name` and all subdirectories from the watcher
func removePathsWithPrefix(watcher *fsnotify.Watcher, name string) error {
	// if the path is "", which happens on linux because we are still watching individual files there,
	// ignore, because we would otherwise remove all the watched prefixes.
	if name == "" {
		log.Debug().Msg("Ignoring empty prefixes")
		return nil
	}
	// we do the "recursion" by checking the watchlist of the watcher for all watched directories
	// that has name as prefix
	watchlist := watcher.WatchList()
	log.Debug().Strs("watchlist", watchlist).Str("name", name).Msg("Removing paths with prefix")
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
