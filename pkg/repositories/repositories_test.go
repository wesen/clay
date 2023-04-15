package repositories

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEmptyNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 0)
}

type TestCommand struct {
	description *cmds.CommandDescription
}

func (t *TestCommand) Description() *cmds.CommandDescription {
	return t.description
}

func makeCommand(parents []string, name string) cmds.Command {
	description := &cmds.CommandDescription{
		Name:    name,
		Parents: parents,
	}
	return &TestCommand{description}
}

func getNames(cmds_ []cmds.Command) []string {
	names := make([]string, len(cmds_))
	for i, c := range cmds_ {
		names[i] = c.Description().Name
	}
	return names
}

func TestSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{makeCommand([]string{}, "test")}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd := makeCommand([]string{}, "test")
	node.InsertCommand(cmd.Description().Parents, cmd)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddRemoveSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd := makeCommand([]string{}, "test")
	node.InsertCommand(cmd.Description().Parents, cmd)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)

	removedCommands := node.Remove([]string{})
	require.Len(t, removedCommands, 1)
	assert.Equal(t, "test", removedCommands[0].Description().Name)

	commands = node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 0)
}

func TestAddTwoCommands(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := makeCommand([]string{}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 2)

	names := getNames(commands)
	assert.Contains(t, names, "test", "test2")

	// remove test
	removedCommands := node.Remove([]string{"test"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test2", commands[0].Description().Name)

	// remove test2
	removedCommands = node.Remove([]string{"test2"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 0)
}

func TestQueryEmptySubbranch(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{"test"}, true)
	assert.Len(t, commands, 0)

	cmd1 := makeCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := makeCommand([]string{}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands = node.CollectCommands([]string{}, true)
	require.Len(t, commands, 2)

	commands = node.CollectCommands([]string{"test"}, true)
	assert.Len(t, commands, 0)

	commands = node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 2)
}

func TestAddCommandDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	assert.Len(t, node.Children, 1)
	aNode := node.Children["a"]
	assert.Len(t, aNode.Children, 1)
	bNode := aNode.Children["b"]
	assert.Len(t, bNode.Children, 1)
	cNode := bNode.Children["c"]
	assert.Len(t, cNode.Children, 0)
	assert.Len(t, cNode.Commands, 1)

	cmd2 := makeCommand([]string{"a", "b", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)
	assert.Len(t, node.Children, 1)
	aNode = node.Children["a"]
	assert.Len(t, aNode.Children, 1)
	bNode = aNode.Children["b"]
	assert.Len(t, bNode.Children, 1)
	cNode = bNode.Children["c"]
	assert.Len(t, cNode.Children, 0)
	assert.Len(t, cNode.Commands, 2)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 2)
	allNames := getNames(commands)
	assert.Contains(t, allNames, "test", "test2")

	commands = node.CollectCommands([]string{"a"}, true)
	require.Len(t, commands, 2)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test", "test2")
}

func TestAddCommandMultipleDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := makeCommand([]string{"a", "b1", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)
	cmd3 := makeCommand([]string{"a", "b2", "c"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)
	cmd4 := makeCommand([]string{"a", "b", "c"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)
	cmd5 := makeCommand([]string{"a", "b1", "c"}, "test5")
	node.InsertCommand(cmd5.Description().Parents, cmd5)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 2)
	allNames := getNames(commands)
	assert.Contains(t, allNames, "test", "test4")

	commands = node.CollectCommands([]string{"a", "b1", "c"}, true)
	require.Len(t, commands, 2)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test2", "test5")

	commands = node.CollectCommands([]string{"a", "b2", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test3", commands[0].Description().Name)

	commands = node.CollectCommands([]string{"a"}, false)
	assert.Len(t, commands, 0)

	commands = node.CollectCommands([]string{"a"}, true)
	require.Len(t, commands, 5)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test", "test2", "test3", "test4", "test5")
}

func TestRemoveCommandDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := makeCommand([]string{"a", "b", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	removedCommands := node.Remove([]string{"a", "b", "c", "test"})
	assert.Len(t, removedCommands, 1)
	assert.Equal(t, "test", removedCommands[0].Description().Name)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test2", commands[0].Description().Name)

	removedCommands = node.Remove([]string{"a", "b", "c", "test2"})
	assert.Len(t, removedCommands, 1)
	assert.Equal(t, "test2", removedCommands[0].Description().Name)

	commands = node.CollectCommands([]string{"a", "b", "c"}, true)
	assert.Len(t, commands, 0)
}

func TestRemoveCommandMultipleDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := makeCommand([]string{"a", "b1", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)
	cmd3 := makeCommand([]string{"a", "b2", "c"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)
	cmd4 := makeCommand([]string{"a", "b", "c"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)
	cmd5 := makeCommand([]string{"a", "b1", "c"}, "test5")
	node.InsertCommand(cmd5.Description().Parents, cmd5)

	removedCommands := node.Remove([]string{"a", "b", "c", "test"})
	assert.Len(t, removedCommands, 1)
	assert.Equal(t, "test", removedCommands[0].Description().Name)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test4", commands[0].Description().Name)

	removedCommands = node.Remove([]string{"a", "b1", "c", "test2"})
	assert.Len(t, removedCommands, 1)
	assert.Equal(t, "test2", removedCommands[0].Description().Name)

	commands = node.CollectCommands([]string{"a", "b1", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test5", commands[0].Description().Name)

	removedCommands = node.Remove([]string{"a", "b2", "c", "test3"})
	assert.Len(t, removedCommands, 1)
	assert.Equal(t, "test3", removedCommands[0].Description().Name)

	commands = node.CollectCommands([]string{"a", "b2", "c"}, true)
	assert.Len(t, commands, 0)

	commands = node.CollectCommands([]string{"a"}, true)
	assert.Len(t, commands, 2)
	allNames := []string{}
	for _, cmd := range commands {
		allNames = append(allNames, cmd.Description().Name)
	}
	assert.Contains(t, allNames, "test4", "test5")
}
