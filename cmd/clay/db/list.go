package db

import (
	"context"
	sql2 "github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/jmoiron/sqlx"
)

func getDBCommandCommandOptions() ([]cmds.CommandDescriptionOption, error) {
	glazeParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	sqlParameterLayer, err := sql2.NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, err
	}
	dbtParameterLayer, err := sql2.NewDbtParameterLayer()
	if err != nil {
		return nil, err
	}

	return []cmds.CommandDescriptionOption{
		cmds.WithLayers(glazeParameterLayer, sqlParameterLayer, dbtParameterLayer),
	}, nil
}

type ListCommandsCommand struct {
	*cmds.CommandDescription
}

func NewListCommandsCommand(options ...cmds.CommandDescriptionOption) (*ListCommandsCommand, error) {
	dbCommandOptions, err := getDBCommandCommandOptions()
	if err != nil {
		return nil, err
	}

	options_ := append(dbCommandOptions, options...)
	options_ = append(options_,
		cmds.WithShort("List all available commands in a database"),
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
				parameters.WithHelp("The type of commands to list"),
				parameters.WithRequired(false),
			),
		),
	)

	return &ListCommandsCommand{
		CommandDescription: cmds.NewCommandDescription("list-commands", options_...),
	}, nil
}

func (D *ListCommandsCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	db, err := sql2.OpenDatabaseFromDefaultSqlConnectionLayer(parsedLayers)
	if err != nil {
		return err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	t := sql2.CreateTemplate(ctx, map[string]string{}, ps, db)
	t_, err := t.Parse("SELECT * FROM {{.table}}")
	if err != nil {
		return err
	}
	s, err := templating.RenderTemplate(t_, ps)
	if err != nil {
		return err
	}

	err = sql2.RunNamedQueryIntoGlaze(ctx, db, s, ps, gp)
	if err != nil {
		return err
	}

	// TODO(manuel, 2023-08-06) Here we could parse the yaml with the RawCommandLoader and use the same output format as the other command listers
	return nil
}
