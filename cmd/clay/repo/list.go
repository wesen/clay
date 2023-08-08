package repo

import (
	"context"
	cmds2 "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/clay/pkg/repositories/fs"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
)

type ListCommand struct {
	*cmds.CommandDescription
}

func NewListCommand(options ...cmds.CommandDescriptionOption) (*ListCommand, error) {
	glazeParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	options = append(options,
		cmds.WithShort("Import a command directory or individual files into a database"),
		cmds.WithFlags(),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"inputs",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("The command directory or individual files to import"),
				parameters.WithRequired(true),
			),
		),
		cmds.WithLayers(glazeParameterLayer),
	)

	return &ListCommand{
		CommandDescription: cmds.NewCommandDescription("list", options...),
	}, nil
}

func (c *ListCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	inputs := ps["inputs"].([]string)
	commands, err := fs.LoadCommandsFromInputs(&cmds2.RawCommandLoader{}, inputs)
	if err != nil {
		return err
	}

	err2 := cmds2.ListCommandsIntoProcessor(ctx, commands, gp)
	if err2 != nil {
		return err2
	}

	return nil
}
