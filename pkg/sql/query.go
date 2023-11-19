package sql

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func RunQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters []interface{},
	gp middlewares.Processor) error {

	// use a prepared statement so that when using mysql, we get native types back
	stmt, err := db.PreparexContext(dbContext, query)
	if err != nil {
		return errors.Wrapf(err, "Could not prepare query: %s", query)
	}

	rows, err := stmt.QueryxContext(dbContext, parameters...)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(dbContext, rows, gp)
}

func RunNamedQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters map[string]interface{},
	gp middlewares.Processor) error {

	// use a statement so that when using mysql, we get native types back
	stmt, err := db.PrepareNamedContext(dbContext, query)
	if err != nil {
		return errors.Wrapf(err, "Could not prepare query: %s", query)
	}

	rows, err := stmt.QueryxContext(dbContext, parameters)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(dbContext, rows, gp)
}

func processQueryResults(ctx context.Context, rows *sqlx.Rows, gp middlewares.Processor) error {
	// we need a way to order the columns
	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrapf(err, "Could not get columns")
	}

	for rows.Next() {
		m := map[string]interface{}{}
		row := types.NewRow()
		err = rows.MapScan(m)
		if err != nil {
			return errors.Wrapf(err, "Could not scan row")
		}

		for _, col := range cols {
			if v, ok := m[col]; ok {
				switch v := v.(type) {
				case []byte:
					row.Set(col, string(v))
				default:
					row.Set(col, v)
				}
			}
		}

		err = gp.AddRow(ctx, row)
		if err != nil {
			return errors.Wrapf(err, "Could not process input object")
		}
	}

	return nil
}

func RunQuery(
	ctx context.Context,
	subQueries map[string]string,
	query string,
	args []interface{},
	ps map[string]interface{},
	db *sqlx.DB,
) (string, *sqlx.Rows, error) {
	if db == nil {
		return "", nil, errors.New("No database connection")
	}

	ps2 := map[string]interface{}{}

	for k, v := range ps {
		ps2[k] = v
	}
	// args is k, v, k, v, k, v
	if len(args)%2 != 0 {
		return "", nil, errors.Errorf("Could not run query: %s", query)
	}
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			return "", nil, errors.Errorf("Could not run query: %s", query)
		}
		ps2[k] = args[i+1]
	}

	t2 := CreateTemplate(ctx, subQueries, ps2, db)
	t, err := t2.Parse(query)
	if err != nil {
		return "", nil, err
	}

	query_, err := templating.RenderTemplate(t, ps2)
	if err != nil {
		return query_, nil, err
	}

	stmt, err := db.PreparexContext(ctx, query_)
	if err != nil {
		return query_, nil, err
	}

	rows, err := stmt.QueryxContext(ctx)
	if err != nil {
		return query_, nil, err
	}

	return query_, rows, err
}

// TODO(manuel, 2023-11-19) Actually return bound arguments,
// probably by adding a funcmap function that one can pipe variable into (but how do we get the name?
// Maybe this is possible in jinja? Other option is to replace the parameters we push into the template and wrap them)

// TODO(manuel, 2023-11-19) Document this section of clay

func RenderQuery(
	ctx context.Context,
	db *sqlx.DB,
	query string,
	subQueries map[string]string,
	ps map[string]interface{},
) (string, error) {
	t2 := CreateTemplate(ctx, subQueries, ps, db)

	t, err := t2.Parse(query)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse query template")
	}

	ret, err := templating.RenderTemplate(t, ps)
	if err != nil {
		return "", errors.Wrap(err, "Could not render query template")
	}

	ret = CleanQuery(ret)
	return ret, nil
}
