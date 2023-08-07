package db

import (
	"context"
	cmds2 "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/clay/pkg/repositories/fs"
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/jmoiron/sqlx"
)

type ImportCommand struct {
	*cmds.CommandDescription
}

func NewImportCommand(options ...cmds.CommandDescriptionOption) (*ImportCommand, error) {
	dbCommandOptions, err := getDBCommandCommandOptions()
	if err != nil {
		return nil, err
	}

	options_ := append(dbCommandOptions, options...)
	options_ = append(options_,
		cmds.WithShort("Import a command directory or individual files into a database"),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"table",
				parameters.ParameterTypeString,
				parameters.WithHelp("The table to list commands for"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"type",
				parameters.ParameterTypeString,
				parameters.WithHelp("The type of commands to import"),
				parameters.WithRequired(true),
			),
		),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"inputs",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("The command directory or individual files to import"),
				parameters.WithRequired(true),
			),
		),
	)

	return &ImportCommand{
		CommandDescription: cmds.NewCommandDescription("import", options_...),
	}, nil
}

func (D *ImportCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	db, err := sql.OpenDatabaseFromDefaultSqlConnectionLayer(parsedLayers)
	if err != nil {
		return err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	inputs := ps["inputs"].([]string)
	commands, err := fs.LoadCommandsFromInputs(&cmds2.RawCommandLoader{}, inputs)
	_ = commands

	return nil
}
