package watcher

import (
	"context"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"os"
	"testing"
	"time"
)

var tempDir string

func TestMain(m *testing.M) {
	// set logging level to debug
	log.Level(zerolog.DebugLevel)

	// create a temporary directory.
	dir, err := os.MkdirTemp("", "clay")
	if err != nil {
		panic(err)
	}
	tempDir = dir
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			panic(err)
		}
	}(dir) // clean up

	m.Run()
}

type watcherTestEvent struct {
	eventType string // "remove" or "write"
	path      string
}

func runWatcherTest(
	t *testing.T,
	main func(ctx context.Context) error,
	expected []watcherTestEvent,
	options ...Option,
) ([]watcherTestEvent, error) {
	eventCh := make(chan watcherTestEvent)
	res := []watcherTestEvent{}
	options_ := append(options, WithWriteCallback(func(path string) error {
		eventCh <- watcherTestEvent{
			eventType: "write",
			path:      path,
		}
		return nil
	}),
		WithRemoveCallback(func(path string) error {
			eventCh <- watcherTestEvent{
				eventType: "remove",
				path:      path,
			}
			return nil
		}))
	w := NewWatcher(options_...)

	eg, ctx := errgroup.WithContext(context.Background())
	ctx2, cancel := context.WithCancel(ctx)

	eg.Go(func() error {
		// wait for max 2 seconds
		timer := time.NewTimer(2 * time.Second)
		for {
			select {
			case <-timer.C:
				return errors.New("timeout")
			case event := <-eventCh:
				res = append(res, event)
				if len(res) == len(expected) {
					cancel()
					return nil
				}
			}
		}
	})

	eg.Go(func() error {
		log.Debug().Msg("starting watcher")
		err := w.Run(ctx2)
		log.Debug().Msg("watcher stopped")
		if err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		timer := time.NewTimer(200 * time.Millisecond)
		<-timer.C
		return main(ctx2)
	})

	err := eg.Wait()
	if err == context.Canceled {
		return res, nil
	}

	assert.Equal(t, expected, res)
	return res, err
}

func TestSimpleFileAddition(t *testing.T) {
	expectedPath := tempDir + "/test.txt"
	_, err := runWatcherTest(
		t,
		func(_ context.Context) error {
			log.Debug().Msgf("creating file %s", expectedPath)
			f, err := os.Create(expectedPath)
			if err != nil {
				return err
			}
			defer func(f *os.File) {
				_ = f.Close()
			}(f)
			return nil
		}, []watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
		},
		WithPaths(tempDir),
	)
	assert.NoError(t, err)
}
