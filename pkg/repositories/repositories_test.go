package repositories

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEmptyNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*cmds.CommandAlias{})
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

func TestSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{makeCommand([]string{}, "test")}, []*cmds.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*cmds.CommandAlias{})
	require.NotNil(t, node)

	cmd := makeCommand([]string{}, "test")
	node.InsertCommand(cmd.Description().Parents, cmd)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddRemoveSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*cmds.CommandAlias{})
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
	node := NewTrieNode([]cmds.Command{}, []*cmds.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := makeCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := makeCommand([]string{}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 2)

	assert.Equal(t, "test", commands[0].Description().Name)
	assert.Equal(t, "test2", commands[1].Description().Name)

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

func TestAddCommandDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*cmds.CommandAlias{})
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
	assert.Equal(t, "test", commands[0].Description().Name)
	assert.Equal(t, "test2", commands[1].Description().Name)

	commands = node.CollectCommands([]string{"a"}, true)
	require.Len(t, commands, 2)
	assert.Equal(t, "test", commands[0].Description().Name)
	assert.Equal(t, "test2", commands[1].Description().Name)
}
