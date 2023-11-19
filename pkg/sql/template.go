package sql

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"strings"
	"text/template"
	"time"
)

// TODO(manuel, 2023-04-23) These should be moved to the templating helpers in  glazed
func sqlEscape(value string) string {
	return strings.Replace(value, "'", "''", -1)
}

func sqlString(value string) string {
	return fmt.Sprintf("'%s'", value)
}

func sqlStringLike(value string) string {
	return fmt.Sprintf("'%%%s%%'", sqlEscape(value))
}

func sqlStringIn(values interface{}) (string, error) {
	strList, ok := cast.CastList2[string, interface{}](values)
	if !ok {
		return "", fmt.Errorf("could not cast %v to []string", values)
	}
	return fmt.Sprintf("'%s'", strings.Join(strList, "','")), nil
}

func sqlIn(values []interface{}) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strValues, ",")
}

func sqlIntIn(values interface{}) string {
	v_, ok := cast.CastInterfaceToIntList[int64](values)
	if !ok {
		return ""
	}
	strValues := make([]string, len(v_))
	for i, v := range v_ {
		strValues[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(strValues, ",")
}

func sqlDate(date interface{}) (string, error) {
	switch v := date.(type) {
	case string:
		parsedDate, err := parameters.ParseDate(v)
		if err != nil {
			return "", err
		}
		return "'" + parsedDate.Format("2006-01-02") + "'", nil
	case time.Time:
		return "'" + v.Format("2006-01-02") + "'", nil
	default:
		return "", fmt.Errorf("could not parse date %v", date)
	}
}

func sqlDateTime(date interface{}) (string, error) {
	switch v := date.(type) {
	case string:
		parsedDate, err := parameters.ParseDate(v)
		if err != nil {
			return "", err
		}
		return "'" + parsedDate.Format("2006-01-02 15:03:04") + "'", nil
	case time.Time:
		return "'" + v.Format("2006-01-02 15:02:03") + "'", nil
	default:
		return "", fmt.Errorf("could not parse date %v", date)
	}
}

func sqlLike(value string) string {
	return "'%" + value + "%'"
}

// TODO(manuel, 2023-11-19) Wrap this in a templating class that can accept additional funcmaps
// (and maybe more templating functionality)

func CreateTemplate(
	ctx context.Context,
	subQueries map[string]string,
	ps map[string]interface{},
	db *sqlx.DB,
) *template.Template {
	t2 := templating.CreateTemplate("query").
		Funcs(templating.TemplateFuncs).
		Funcs(template.FuncMap{
			"sqlStringIn":   sqlStringIn,
			"sqlStringLike": sqlStringLike,
			"sqlIntIn":      sqlIntIn,
			"sqlIn":         sqlIn,
			"sqlDate":       sqlDate,
			"sqlDateTime":   sqlDateTime,
			"sqlLike":       sqlLike,
			"sqlString":     sqlString,
			"sqlEscape":     sqlEscape,
			"subQuery": func(name string) (string, error) {
				s, ok := subQueries[name]
				if !ok {
					return "", errors.Errorf("Subquery %s not found", name)
				}
				return s, nil
			},
			"sqlSlice": func(query string, args ...interface{}) ([]interface{}, error) {
				_, rows, err := RunQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					// TODO(manuel, 2023-03-27) This nesting of errors in nested templates becomes quite unpalatable
					// This is what can be output for just one level deep:
					//
					// Error: Could not generate query: template: query:1:13: executing "query" at <sqlColumn (subQuery "post_types")>: error calling sqlColumn: Could not run query: SELECT post_type
					// FROM wp_posts
					// GROUP BY post_type
					// ORDER BY post_type
					// : Error 1146 (42S02): Table 'ttc_analytics.wp_posts' doesn't exist
					// exit status 1
					//
					// Make better error messages:
					return nil, errors.Wrapf(err, "Could not run query: %s", query)
				}
				defer func(rows *sqlx.Rows) {
					_ = rows.Close()
				}(rows)

				ret := []interface{}{}

				for rows.Next() {
					ret_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", query)
					}

					row := make([]interface{}, len(ret_))
					for i, v := range ret_ {
						row[i] = sqlEltToTemplateValue(v)
					}

					ret = append(ret, row)
				}

				return ret, nil
			},
			"sqlColumn": func(query string, args ...interface{}) ([]interface{}, error) {
				renderedQuery, rows, err := RunQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer func(rows *sqlx.Rows) {
					_ = rows.Close()
				}(rows)

				ret := make([]interface{}, 0)
				for rows.Next() {
					rows_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					if len(rows_) != 1 {
						return nil, errors.Errorf("Expected 1 column, got %d", len(rows_))
					}
					elt := rows_[0]

					v := sqlEltToTemplateValue(elt)

					ret = append(ret, v)
				}

				return ret, nil
			},
			"sqlSingle": func(query string, args ...interface{}) (interface{}, error) {
				renderedQuery, rows, err := RunQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer func(rows *sqlx.Rows) {
					_ = rows.Close()
				}(rows)

				ret := make([]interface{}, 0)
				if rows.Next() {
					rows_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					if len(rows_) != 1 {
						return nil, errors.Errorf("Expected 1 column, got %d", len(rows_))
					}

					ret = append(ret, rows_[0])
				}

				if rows.Next() {
					return nil, errors.Errorf("Expected 1 row, got more")
				}

				if len(ret) == 0 {
					return nil, nil
				}

				if len(ret) > 1 {
					return nil, errors.Errorf("Expected 1 row, got %d", len(ret))
				}

				return sqlEltToTemplateValue(ret[0]), nil
			},
			"sqlMap": func(query string, args ...interface{}) (interface{}, error) {
				renderedQuery, rows, err := RunQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer func(rows *sqlx.Rows) {
					_ = rows.Close()
				}(rows)

				ret := []map[string]interface{}{}

				for rows.Next() {
					ret_ := make(map[string]interface{})
					err = rows.MapScan(ret_)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					row := make(map[string]interface{})
					for k, v := range ret_ {
						row[k] = sqlEltToTemplateValue(v)
					}

					ret = append(ret, row)
				}

				return ret, nil
			},
		})

	return t2
}

func sqlEltToTemplateValue(elt interface{}) interface{} {
	switch v := elt.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func CleanQuery(query string) string {
	// remove all empty whitespace lines
	v := filter(
		strings.Split(query, "\n"),
		func(s string) bool {
			return strings.TrimSpace(s) != ""
		},
	)
	query = strings.Join(
		smap(v, func(s string) string {
			return strings.TrimRight(s, " \t")
		}),
		"\n",
	)

	return query
}

func smap(strs []string, f func(s string) string) []string {
	ret := make([]string, len(strs))
	for i, s := range strs {
		ret[i] = f(s)
	}
	return ret
}

func filter(strs []string, f func(s string) bool) []string {
	ret := make([]string, 0, len(strs))
	for _, s := range strs {
		if f(s) {
			ret = append(ret, s)
		}
	}
	return ret
}
