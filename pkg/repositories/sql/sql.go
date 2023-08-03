package sql

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"strings"
	"time"
)

type Repo struct {
	db            *sqlx.DB
	tableName     string
	commandLoader loaders.YAMLCommandLoader
}

func NewRepo(connection string, tableName string, loader loaders.YAMLCommandLoader) (*Repo, error) {
	db, err := sqlx.Open("postgres", connection)

	if err != nil {
		return nil, err
	}

	return &Repo{
		db:            db,
		tableName:     tableName,
		commandLoader: loader, // Initialize with the loader
	}, nil
}

type Command struct {
	Id          int64
	Name        string
	Type        string
	Body        string
	Version     int64
	CreatedAt   time.Time
	CreatedBy   string
	Description string
}

func (r *Repo) ListCommands(ctx context.Context, prefixes []string, recurse bool) ([]*Command, error) {
	commands := make([]*Command, 0)

	for _, prefix := range prefixes {
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, command_name, command_type, command_body, version_number, created_at, created_by, description FROM `+r.tableName+` AS t1 
            INNER JOIN (
                SELECT command_name, MAX(version_number) AS max_version 
                FROM `+r.tableName+`
                GROUP BY command_name
            ) AS t2 
            ON t1.command_name = t2.command_name AND t1.version_number = t2.max_version 
            WHERE t1.command_name LIKE $1`, prefix+"%")

		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var cmd Command
			err = rows.Scan(&cmd.Id, &cmd.Name, &cmd.Type, &cmd.Body, &cmd.Version, &cmd.CreatedAt, &cmd.CreatedBy, &cmd.Description)
			if err != nil {
				return nil, err
			}

			separateCommands := strings.Split(cmd.Name, "/")
			if recurse || len(separateCommands) == 1 || separateCommands[0] == prefix {
				commands = append(commands, &cmd)
			}
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	return commands, nil
}

func (r *Repo) CollectCommands(prefixes []string, recurse bool) ([]cmds.Command, error) {
	var ctx context.Context
	var commands []*Command

	// Use context for cancellation and timeouts, passing it through function calls.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	commands, err := r.ListCommands(ctx, prefixes, recurse)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list commands")
	}

	transformedCommands := []cmds.Command{}
	// Use commandLoader to load the cmd.Command from Command.Body
	for _, cmd := range commands {
		reader := strings.NewReader(cmd.Body)
		cmds_, err := r.commandLoader.LoadCommandFromYAML(reader)
		if err != nil {
			// Try to load as alias if fails to load as command
			aliases, errAlias := r.commandLoader.LoadCommandAliasFromYAML(reader)
			if errAlias != nil {
				return nil, errors.Wrap(errAlias, "failed to load command and alias")
			}

			for _, alias := range aliases {
				transformedCommands = append(transformedCommands, alias)
			}
		} else {
			// if command is loaded successfully, append to result.
			transformedCommands = append(transformedCommands, cmds_...)
		}
	}

	return transformedCommands, nil
}

func (r *Repo) ListIntoGlaze(ctx context.Context, gp middlewares.Processor, prefixes []string, recurse bool) error {
	cmds, err := r.ListCommands(ctx, prefixes, recurse)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		err = gp.AddRow(ctx, types.NewRowFromStruct(cmd, false))
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repo) InsertCommand(cmd *Command) error {
	cmd.Version += 1

	_, err := r.db.Exec(`INSERT INTO `+r.tableName+` (command_name, command_type, command_body, version_number, created_at, created_by, description) VALUES($1, $2, $3, $4, $5, $6, $7)`,
		cmd.Name, cmd.Type, cmd.Body, cmd.Version, cmd.CreatedAt, cmd.CreatedBy, cmd.Description)

	return err
}

func (r *Repo) DeleteCommand(cmdName string) error {
	_, err := r.db.Exec(`DELETE FROM `+r.tableName+` WHERE command_name = $1`, cmdName)
	return err
}

func (r *Repo) UpdateCommand(cmd *Command) error {
	// increment version and insert as a new command
	return r.InsertCommand(cmd)
}

func (r *Repo) CreateTable(ctx context.Context) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			command_name TEXT NOT NULL,
			command_type TEXT NOT NULL,
			command_body TEXT,
			version_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE,
			created_by TEXT,
			description TEXT
		);`, r.tableName)
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", r.tableName, err)
	}
	return nil
}
