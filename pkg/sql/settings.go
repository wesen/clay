package sql

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed "flags/sql-connection.yaml"
var connectionFlagsYaml []byte

type ConnectionParameterLayer struct {
	layers.ParameterLayerImpl `yaml:",inline"`
}

func NewSqlConnectionParameterLayer(
	options ...layers.ParameterLayerOptions,
) (*ConnectionParameterLayer, error) {
	layer, err := layers.NewParameterLayerFromYAML(connectionFlagsYaml, options...)
	if err != nil {
		return nil, err
	}
	ret := &ConnectionParameterLayer{}
	ret.ParameterLayerImpl = *layer

	return ret, nil
}

func (cp *ConnectionParameterLayer) ParseFlagsFromCobraCommand(cmd *cobra.Command) (map[string]interface{}, error) {
	// actually hijack and load everything from viper instead of cobra...
	ps, err := parameters.GatherFlagsFromViper(cp.Flags, false, cp.Prefix)
	if err != nil {
		return nil, err
	}

	// now load from flag overrides
	ps2, err := parameters.GatherFlagsFromCobraCommand(cmd, cp.Flags, true, cp.Prefix)
	if err != nil {
		return nil, err
	}
	for k, v := range ps2 {
		ps[k] = v
	}

	return ps, nil
}

//go:embed "flags/dbt.yaml"
var dbtFlagsYaml []byte

type DbtParameterLayer struct {
	layers.ParameterLayerImpl `yaml:",inline"`
}

func NewDbtParameterLayer(
	options ...layers.ParameterLayerOptions,
) (*DbtParameterLayer, error) {
	ret, err := layers.NewParameterLayerFromYAML(dbtFlagsYaml, options...)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize dbt parameter layer")
	}
	return &DbtParameterLayer{
		ParameterLayerImpl: *ret,
	}, nil
}

func (d *DbtParameterLayer) ParseFlagsFromCobraCommand(cmd *cobra.Command) (map[string]interface{}, error) {
	// actually hijack and load everything from viper instead of cobra...
	ps := make(map[string]interface{})

	for _, f := range d.Flags {
		//exhaustive:ignore
		switch f.Type {
		case parameters.ParameterTypeString:
			v := viper.GetString(d.Prefix + f.Name)
			ps[f.Name] = v
		case parameters.ParameterTypeInteger:
			v := viper.GetInt(d.Prefix + f.Name)
			ps[f.Name] = v
		case parameters.ParameterTypeBool:
			v := viper.GetBool(d.Prefix + f.Name)
			ps[f.Name] = v
		default:
			return nil, errors.Errorf("Unknown DBT parameter type %s for flag %s", f.Type, f.Name)
		}
	}

	// now load from flag overrides
	ps2, err := parameters.GatherFlagsFromCobraCommand(cmd, d.Flags, true, d.Prefix)
	if err != nil {
		return nil, err
	}
	for k, v := range ps2 {
		ps[k] = v
	}

	return ps, nil
}

func OpenDatabaseFromDefaultSqlConnectionLayer(
	parsedLayers map[string]*layers.ParsedParameterLayer,
) (*sqlx.DB, error) {
	return OpenDatabaseFromSqlConnectionLayer(parsedLayers, "sql-connection", "dbt")
}

func OpenDatabaseFromSqlConnectionLayer(
	parsedLayers map[string]*layers.ParsedParameterLayer,
	sqlConnectionLayerName string,
	dbtLayerName string,
) (*sqlx.DB, error) {
	sqlConnectionLayer, ok := parsedLayers[sqlConnectionLayerName]
	if !ok {
		return nil, errors.New("No sql-connection layer found")
	}
	dbtLayer, ok := parsedLayers[dbtLayerName]
	if !ok {
		return nil, errors.New("No dbt layer found")
	}

	config, err2 := NewConfigFromParsedLayers(sqlConnectionLayer, dbtLayer)
	if err2 != nil {
		return nil, err2
	}
	return config.Connect()
}
