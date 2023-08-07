package db

import (
	"context"
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/jmoiron/sqlx"
)

type CreateRepoCommand struct {
	*cmds.CommandDescription
}

func NewCreateRepoCommand(options ...cmds.CommandDescriptionOption) (*CreateRepoCommand, error) {
	dbCommandOptions, err := getDBCommandCommandOptions()
	if err != nil {
		return nil, err
	}

	options_ := append(dbCommandOptions, options...)
	options_ = append(options_,
		cmds.WithShort("Create a new table to store commands"),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"table",
				parameters.ParameterTypeString,
				parameters.WithHelp("The table to create"),
				parameters.WithRequired(true),
			),
		),
	)

	return &CreateRepoCommand{
		CommandDescription: cmds.NewCommandDescription("create-repo", options_...),
	}, nil
}

func (D *CreateRepoCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	db, err := sql.OpenDatabaseFromDefaultSqlConnectionLayer(parsedLayers)
	if err != nil {
		return err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	repo := sql.NewRepo(db, ps["table"].(string), nil)
	err = repo.CreateTable(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Created table " + ps["table"].(string))
	return nil
}
