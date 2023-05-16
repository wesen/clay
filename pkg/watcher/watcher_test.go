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

func TestSimpleFileAddition(t *testing.T) {
	addedCh := make(chan string)

	w := NewWatcher(WithPaths(tempDir), WithWriteCallback(func(path string) error {
		log.Debug().Str("path", path).Msg("received path")
		addedCh <- path
		return nil
	}))

	expectedPath := tempDir + "/test.txt"

	eg, ctx := errgroup.WithContext(context.Background())
	ctx2, cancel := context.WithCancel(ctx)

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
		defer cancel()

		log.Debug().Msg("waiting for file to be added")

		// wait for 2 seconds before failing, in case nothing happens
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		log.Debug().Msg("waiting for file to be added")

		select {
		case <-timer.C:
			log.Debug().Msg("timed out waiting for file to be added")
			assert.Fail(t, "timed out waiting for file to be added")
			return errors.New("timed out waiting for file to be added")
		case path := <-addedCh:
			log.Debug().Msgf("received path %s", path)
			assert.Equal(t, expectedPath, path)
			if path != expectedPath {
				return errors.New("received path does not match expected path")
			}
			return nil
		case <-ctx2.Done():
			return nil
		}
	})

	eg.Go(func() error {
		// wait for 500 ms before creating file, to have the watcher settle
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			log.Debug().Msgf("creating file %s", expectedPath)
			f, err := os.Create(expectedPath)
			if err != nil {
				return err
			}
			defer f.Close()
			return nil
		case <-ctx2.Done():
			return nil
		}
	})

	err := eg.Wait()
	if err == context.Canceled {
		return
	}
	assert.NoError(t, err)
}
