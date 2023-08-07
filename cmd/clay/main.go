package main

import (
	"embed"
	"github.com/go-go-golems/clay/cmd/clay/db"
	"github.com/go-go-golems/clay/cmd/clay/repo"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
)

//go:embed doc/*
var docFS embed.FS

func createRootCmd() *cobra.Command {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	cobra.CheckErr(err)

	rootCmd := &cobra.Command{
		Use:   "clay",
		Short: "clay is a CLI tool for managing GO GO GOLEMS business applications",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := clay.InitLogger()
			cobra.CheckErr(err)
		},
	}

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("clay", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	return rootCmd
}

func main() {
	rootCmd := createRootCmd()

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}
	rootCmd.AddCommand(dbCmd)

	listCommandsCommand, err := db.NewListCommandsCommand()
	cobra.CheckErr(err)
	cmd, err := cli.BuildCobraCommandFromGlazeCommand(listCommandsCommand)
	cobra.CheckErr(err)
	dbCmd.AddCommand(cmd)

	createRepoCommand, err := db.NewCreateRepoCommand()
	cobra.CheckErr(err)
	cmd, err = cli.BuildCobraCommandFromBareCommand(createRepoCommand)
	cobra.CheckErr(err)
	dbCmd.AddCommand(cmd)

	repoCmd := &cobra.Command{
		Use:   "repo",
		Short: "Repository management commands",
	}
	rootCmd.AddCommand(repoCmd)

	listRepoCommandsCommand, err := repo.NewListCommand()
	cobra.CheckErr(err)
	cmd, err = cli.BuildCobraCommandFromGlazeCommand(listRepoCommandsCommand)
	cobra.CheckErr(err)
	repoCmd.AddCommand(cmd)

	err = rootCmd.Execute()
	cobra.CheckErr(err)
}
