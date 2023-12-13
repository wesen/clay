package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"strings"
)

type RawCommandLoader struct{}

var _ loaders.FileCommandLoader = (*RawCommandLoader)(nil)

type RawCommand struct {
	*cmds.CommandDescription
	YAMLContent map[string]interface{}
	Content     []byte
}

func (r *RawCommand) ToYAML(w io.Writer) error {
	return yaml.NewEncoder(w).Encode(r.YAMLContent)
}

func (r *RawCommandLoader) LoadCommandsFromReader(
	s io.Reader,
	options []cmds.CommandDescriptionOption,
	aliasOptions []alias.Option,
) ([]cmds.Command, error) {
	// first parse the CommandDescription
	return loaders.LoadCommandOrAliasFromReader(
		s,
		loadRawCommand,
		options,
		aliasOptions,
	)
}

func loadRawCommand(
	s io.Reader,
	options []cmds.CommandDescriptionOption,
	_ []alias.Option,
) ([]cmds.Command, error) {
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

	for _, option := range options {
		option(description)
	}

	return []cmds.Command{
		&RawCommand{
			CommandDescription: description,
			YAMLContent:        allYaml,
			Content:            buf,
		},
	}, nil
}

func (r2 *RawCommandLoader) IsFileSupported(f fs.FS, fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

func NewRawCommandLoader() loaders.FileCommandLoader {
	return &RawCommandLoader{}
}
