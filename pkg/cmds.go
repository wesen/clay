package pkg

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/fs"
	"os"
)

type EmbeddedCommandLocation struct {
	FS      fs.FS
	Name    string
	Root    string
	DocRoot string
}

type CommandLocations struct {
	Embedded     []EmbeddedCommandLocation
	Repositories []string
}

func (c *CommandLocations) LoadCommands(
	loader glazed_cmds.FSCommandLoader,
	helpSystem *help.HelpSystem,
	rootCmd *cobra.Command,
) ([]glazed_cmds.Command, []*glazed_cmds.CommandAlias, error) {
	// Load the variables from the environment

	log.Debug().
		Str("config", viper.ConfigFileUsed()).
		Msg("Loaded configuration")

	var commands []glazed_cmds.Command
	var aliases []*glazed_cmds.CommandAlias
	for _, e := range c.Embedded {
		commands_, aliases_, err := loader.LoadCommandsFromFS(e.FS, e.Root)
		if err != nil {
			return nil, nil, err
		}
		commands = append(commands, commands_...)
		aliases = append(aliases, aliases_...)

		err = helpSystem.LoadSectionsFromFS(e.FS, e.DocRoot)
		if err != nil {
			return nil, nil, err
		}

	}

	repositoryCommands, repositoryAliases, err := c.loadRepositoryCommands(loader, helpSystem)
	if err != nil {
		return nil, nil, err
	}

	commands = append(commands, repositoryCommands...)
	aliases = append(aliases, repositoryAliases...)

	// here is where i need to set the connection factory and add the sqleton layers
	err = cli.AddCommandsToRootCommand(rootCmd, commands, aliases)
	if err != nil {
		return nil, nil, err
	}

	return commands, aliases, nil
}

func (c *CommandLocations) loadRepositoryCommands(
	loader glazed_cmds.FSCommandLoader,
	helpSystem *help.HelpSystem,
) ([]glazed_cmds.Command, []*glazed_cmds.CommandAlias, error) {

	commands := make([]glazed_cmds.Command, 0)
	aliases := make([]*glazed_cmds.CommandAlias, 0)

	for _, repository := range c.Repositories {
		repository = os.ExpandEnv(repository)

		// check that repository exists and is a directory
		s, err := os.Stat(repository)

		if os.IsNotExist(err) {
			log.Debug().Msgf("Repository %s does not exist", repository)
			continue
		} else if err != nil {
			log.Warn().Msgf("Error while checking directory %s: %s", repository, err)
			continue
		}

		if s == nil || !s.IsDir() {
			log.Warn().Msgf("Repository %s is not a directory", repository)
		} else {
			docDir := fmt.Sprintf("%s/doc", repository)
			commands_, aliases_, err := loader.LoadCommandsFromFS(os.DirFS(repository), ".")
			if err != nil {
				return nil, nil, err
			}
			commands = append(commands, commands_...)
			aliases = append(aliases, aliases_...)

			_, err = os.Stat(docDir)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				log.Debug().Err(err).Msgf("Error while checking directory %s", docDir)
				continue
			}
			err = helpSystem.LoadSectionsFromFS(os.DirFS(docDir), ".")
			if err != nil {
				log.Warn().Err(err).Msgf("Error while loading help sections from directory %s", repository)
				continue
			}
		}
	}
	return commands, aliases, nil
}
