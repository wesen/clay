package fs

import (
	"context"
	"fmt"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (r *Repository) Watch(
	ctx context.Context,
	options ...watcher.Option,
) error {
	if r.readerLoader == nil {
		return fmt.Errorf("no command loader set")
	}

	fs_ := os.DirFS("/")
	options = append(options,
		watcher.WithWriteCallback(func(path string) error {
			log.Debug().Msgf("Loading %s", path)
			f, err := fs_.Open(strings.TrimPrefix(path, "/"))
			if err != nil {
				log.Warn().Str("path", path).Err(err).Msg("could not open file")
				return err
			}
			defer func(f fs.File) {
				_ = f.Close()
			}(f)

			fullPath := path
			// try to strip all r.Directories from path
			// if it's not possible, then just use path
			for _, dir := range r.Directories {
				if strings.HasPrefix(path, dir) {
					path = strings.TrimPrefix(path, dir)
					break
				}
			}
			path = strings.TrimPrefix(path, "/")

			// get directory of file
			parents := loaders.GetParentsFromDir(filepath.Dir(path))
			cmdOptions_ := append(r.cmdOptions,
				cmds.WithSource(fullPath),
				cmds.WithParents(parents...))
			aliasOptions := []alias.Option{
				alias.WithSource(fullPath),
				alias.WithParents(parents...),
			}
			commands, err := r.readerLoader.LoadCommandsFromReader(f, cmdOptions_, aliasOptions)
			if err != nil {
				return err
			}
			r.Add(commands...)
			return nil
		}),
		watcher.WithRemoveCallback(func(path string) error {
			log.Debug().Msgf("Removing %s", path)
			r.Remove([]string{path})
			return nil
		}),
		watcher.WithPaths(r.Directories...),
	)
	w := watcher.NewWatcher(options...)

	err := w.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "could not run watcher for repository: %s", strings.Join(r.Directories, ","))
	}
	return nil
}
