package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"gopkg.in/yaml.v3"
	"io"
)

type RawCommandLoader struct{}

type RawCommand struct {
	*cmds.CommandDescription
	YAMLContent map[string]interface{}
	Content     []byte
}

func (r *RawCommand) ToYAML(w io.Writer) error {
	return yaml.NewEncoder(w).Encode(r.YAMLContent)
}

func (r *RawCommandLoader) LoadCommandFromYAML(s io.Reader, options ...cmds.CommandDescriptionOption) ([]cmds.Command, error) {
	// first parse the CommandDescription
	description := &cmds.CommandDescription{}

	buf, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(buf, description)
	if err != nil {
		return nil, err
	}

	allYaml := map[string]interface{}{}
	err = yaml.Unmarshal(buf, &allYaml)
	if err != nil {
		return nil, err
	}

	return []cmds.Command{
		&RawCommand{
			CommandDescription: description,
			YAMLContent:        allYaml,
			Content:            buf,
		},
	}, nil
}

func (r *RawCommandLoader) LoadCommandAliasFromYAML(s io.Reader, options ...alias.Option) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}
