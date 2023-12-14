package fs

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"os"
)

func LoadCommandsFromInputs(
	commandLoader loaders.FileCommandLoader,
	inputs []string,
) ([]cmds.Command, error) {
	files := []string{}
	directories := []string{}
	for _, input := range inputs {
		// check if is directory
		s, err := os.Stat(input)
		if err != nil {
			return nil, err
		}
		if s.IsDir() {
			directories = append(directories, input)
		} else {
			files = append(files, input)
		}
	}

	fsLoader := loaders.NewFSFileCommandLoader(commandLoader)
	repository := NewRepository(
		WithFSLoader(fsLoader),
		WithReaderCommandLoader(commandLoader),
		WithDirectories(directories),
	)

	err := repository.LoadCommands()
	if err != nil {
		return nil, err
	}

	commands := repository.CollectCommands([]string{}, true)
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		cmds_, err := commandLoader.LoadCommandsFromReader(f, []cmds.CommandDescriptionOption{}, []alias.Option{})
		if err != nil {
			return nil, err
		}

		commands = append(commands, cmds_...)
	}

	return commands, nil
}
