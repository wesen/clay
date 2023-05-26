package repositories

import (
	"context"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
)

func (r *Repository) Watch(
	ctx context.Context,
	options ...watcher.Option,
) error {
	fs := os.DirFS("/")
	options = append(options,
		watcher.WithWriteCallback(func(path string) error {
			log.Debug().Msgf("Loading %s", path)
			f, err := fs.Open(strings.TrimPrefix(path, "/"))
			if err != nil {
				log.Warn().Str("path", path).Err(err).Msg("could not open file")
				return err
			}
			defer f.Close()

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
			commands, err := r.loader.LoadCommandsFromReader(f, cmdOptions_, aliasOptions)
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

	return w.Run(ctx)
}
