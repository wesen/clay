package repositories

import (
	"context"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/rs/zerolog/log"
	"os"
)

func WatchRepositories(
	ctx context.Context,
	loader loaders.FSCommandLoader,
	r *Repository,
	options ...watcher.Option,
) error {
	fs := os.DirFS("/")
	options = append(options,
		watcher.WithWriteCallback(func(path string) error {
			log.Debug().Msgf("Loading %s", path)
			// XXX need to pass CommandDescriptionOption
			commands, aliases, err := loader.LoadCommandsFromFS(fs, path, nil, nil)
			if err != nil {
				return err
			}
			r.Add(commands...)
			aliasCommands, ok := cast.CastList[cmds.Command](aliases)
			if ok {
				r.Add(aliasCommands...)
			}
			return nil
		}),
		watcher.WithRemoveCallback(func(path string) error {
			log.Debug().Msgf("Removing %s", path)
			r.Remove([]string{path})
			return nil
		}))
	w := watcher.NewWatcher(options...)

	return w.Run(ctx)
}
