package main

import (
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"math/rand"
	"time"
)

type TestCommand struct {
	description *cmds.CommandDescription
}

func (t *TestCommand) Description() *cmds.CommandDescription {
	return t.description
}

func MakeTestCommand(parents []string, name string) cmds.Command {
	description := &cmds.CommandDescription{
		Name:    name,
		Parents: parents,
	}
	return &TestCommand{description}
}

func main() {
	// set up trie
	root := repositories.NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})

	// insert commands
	numCommands := 1000
	commands := make([]cmds.Command, numCommands)
	for i := 0; i < numCommands; i++ {
		prefix := MakeRandomPrefix(2)
		cmd := MakeTestCommand(prefix, fmt.Sprintf("cmd-%d", i))
		commands[i] = cmd

		start := time.Now()
		root.InsertCommand(prefix, cmd)
		elapsed := time.Since(start)
		fmt.Printf("Inserted command %d in %s\n", i, elapsed)
	}

	// remove commands
	for i := 0; i < numCommands; i++ {
		cmd := commands[i]

		start := time.Now()
		root.Remove(cmd.Description().Parents)
		elapsed := time.Since(start)
		fmt.Printf("Removed command %d in %s\n", i, elapsed)
	}
}

func MakeRandomPrefix(depth int) []string {
	prefix := make([]string, depth)
	for i := 0; i < depth; i++ {
		// prefix name is random between 1 and 10
		pi := 1 + rand.Intn(10)
		prefix[i] = fmt.Sprintf("prefix-%d", pi)
	}

	return prefix
}
