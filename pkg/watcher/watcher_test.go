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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)

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
	eventCh := make(chan watcherTestEvent, 10)
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
				log.Error().Msg("timeout")
				return errors.New("timeout")
			case event := <-eventCh:
				res = append(res, event)
				if len(res) >= len(expected) {
					log.Debug().
						Int("expectedLength", len(expected)).
						Int("actualLength", len(res)).
						Msg("got all events")
					cancel()
					return nil
				}
			}
		}
	})

	eg.Go(func() error {
		log.Debug().Msg("starting watcher")
		err := w.Run(ctx2)
		log.Debug().Err(err).Msg("watcher stopped")
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
		err = nil
	}
	if err != nil {
		return nil, err
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

func TestSimpleFileRemoval(t *testing.T) {
	expectedPath := tempDir + "/test.txt"
	_, err := runWatcherTest(
		t,
		func(_ context.Context) error {
			log.Debug().Msgf("creating file %s", expectedPath)
			f, err := os.Create(expectedPath)
			if err != nil {
				return err
			}
			_ = f.Close()

			log.Debug().Msgf("removing file %s", expectedPath)
			err = os.Remove(expectedPath)
			if err != nil {
				return err
			}
			return nil
		},
		[]watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "remove",
				path:      expectedPath,
			},
		},
		WithPaths(tempDir),
	)

	assert.NoError(t, err)
}

func TestFilteredFileAdd(t *testing.T) {
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

			log.Debug().Msgf("creating file %s", tempDir+"/test2.doc")
			f, err = os.Create(tempDir + "/test2.doc")
			if err != nil {
				return err
			}
			defer func(f *os.File) {
				_ = f.Close()
			}(f)
			return nil
		},
		[]watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
		},
		WithPaths(tempDir),
		WithMask("**/*.txt"),
	)

	assert.NoError(t, err)
}

func TestTwoWrites(t *testing.T) {
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

			log.Debug().Msgf("writing to file %s", expectedPath)
			_, err = f.WriteString("hello world")
			if err != nil {
				return err
			}
			return nil
		},
		[]watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "write",
				path:      expectedPath,
			},
		},
		WithPaths(tempDir),
	)

	assert.NoError(t, err)
}

func TestRename(t *testing.T) {
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

			log.Debug().Msgf("renaming file %s", expectedPath)
			err = os.Rename(expectedPath, tempDir+"/test2.txt")
			if err != nil {
				return err
			}
			return nil
		},
		[]watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "remove",
				path:      expectedPath,
			},
			{
				eventType: "write",
				path:      tempDir + "/test2.txt",
			},
		},
		WithPaths(tempDir),
	)

	assert.NoError(t, err)
}

func TestRenameFiveTimes(t *testing.T) {
	expectedPath := tempDir + "/test.txt"
	_, err := runWatcherTest(
		t,
		func(_ context.Context) error {
			log.Debug().Msgf("creating file %s", expectedPath)
			f, err := os.Create(expectedPath)
			if err != nil {
				return err
			}
			_ = f.Close()

			intervalDuration := 10 * time.Millisecond
			time.Sleep(intervalDuration)

			for i := 0; i < 3; i++ {
				log.Debug().Msgf("%d - renaming file %s to %s", i, expectedPath, tempDir+"/test2.txt")
				err = os.Rename(expectedPath, tempDir+"/test2.txt")
				if err != nil {
					return err
				}

				time.Sleep(intervalDuration)

				log.Debug().Msgf("%d - renaming file %s to %s", i, tempDir+"/test2.txt", expectedPath)
				err = os.Rename(tempDir+"/test2.txt", expectedPath)
				if err != nil {
					return err
				}

				time.Sleep(intervalDuration)
			}

			log.Debug().Msgf("removing file %s", expectedPath)
			err = os.Remove(expectedPath)
			if err != nil {
				return err
			}

			time.Sleep(intervalDuration)
			return nil
		},
		[]watcherTestEvent{
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "remove",
				path:      expectedPath,
			},
			{
				eventType: "write",
				path:      tempDir + "/test2.txt",
			},
			{
				eventType: "remove",
				path:      tempDir + "/test2.txt",
			},
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "remove",
				path:      expectedPath,
			},
			{
				eventType: "write",
				path:      tempDir + "/test2.txt",
			},
			{
				eventType: "remove",
				path:      tempDir + "/test2.txt",
			},
			{
				eventType: "write",
				path:      expectedPath,
			},
			{
				eventType: "remove",
				path:      expectedPath,
			},
			{
				eventType: "write",
				path:      tempDir + "/test2.txt",
			},
			{
				eventType: "remove",
				path:      tempDir + "/test2.txt",
			},
		},
		WithPaths(tempDir),
	)

	assert.NoError(t, err)
}
