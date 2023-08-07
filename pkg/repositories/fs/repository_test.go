package fs

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
	*cmds.CommandDescription
}

func MakeTestCommand(parents []string, name string) cmds.Command {
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
	node := NewTrieNode([]cmds.Command{MakeTestCommand([]string{}, "test")}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd := MakeTestCommand([]string{}, "test")
	node.InsertCommand(cmd.Description().Parents, cmd)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)

	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestAddRemoveSingleCommandNode(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd := MakeTestCommand([]string{}, "test")
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

	cmd1 := MakeTestCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := MakeTestCommand([]string{}, "test2")
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

func TestAddThreeCommands(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := MakeTestCommand([]string{}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	cmd3 := MakeTestCommand([]string{}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)

	commands := node.CollectCommands([]string{}, true)
	require.Len(t, commands, 3)

	names := getNames(commands)
	assert.Contains(t, names, "test", "test2", "test3")

	// remove test
	removedCommands := node.Remove([]string{"test"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{}, true)
	require.Len(t, commands, 2)
	names = getNames(commands)
	assert.Contains(t, names, "test2", "test3")

	// remove test2
	removedCommands = node.Remove([]string{"test2"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test3", commands[0].Description().Name)

	// remove test3
	removedCommands = node.Remove([]string{"test3"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 0)
}

func TestAddThreeCommandsTwoLevelsDeep(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"test"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := MakeTestCommand([]string{"test"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	cmd3 := MakeTestCommand([]string{"test"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)

	commands := node.CollectCommands([]string{"foobar"}, true)
	require.Len(t, commands, 0)

	commands = node.CollectCommands([]string{"test"}, true)
	require.Len(t, commands, 3)

	names := getNames(commands)
	assert.Contains(t, names, "test", "test2", "test3")

	// remove test
	removedCommands := node.Remove([]string{"test", "test"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{"test"}, true)
	require.Len(t, commands, 2)
	names = getNames(commands)
	assert.Contains(t, names, "test2", "test3")

	// add a command one level deeper
	cmd4 := MakeTestCommand([]string{"test", "test2"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)

	commands = node.CollectCommands([]string{"test"}, true)
	require.Len(t, commands, 3)
	names = getNames(commands)
	assert.Contains(t, names, "test2", "test3", "test4")

	commands = node.CollectCommands([]string{"test"}, true)
	require.Len(t, commands, 3)
	names = getNames(commands)
	assert.Contains(t, names, "test2", "test3", "test")

	commands = node.CollectCommands([]string{"test"}, false)
	require.Len(t, commands, 0)

	// remove test2
	removedCommands = node.Remove([]string{"test", "test2"})
	assert.Equal(t, 2, len(removedCommands))
	names = getNames(removedCommands)
	assert.Contains(t, names, "test2", "test4")

	commands = node.CollectCommands([]string{"test"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test3", commands[0].Description().Name)

	// remove test3
	removedCommands = node.Remove([]string{"test", "test3"})
	assert.Equal(t, 1, len(removedCommands))

	commands = node.CollectCommands([]string{"test"}, true)
	assert.Len(t, commands, 0)
}

func TestQueryEmptySubbranch(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	commands := node.CollectCommands([]string{"test"}, true)
	assert.Len(t, commands, 0)

	cmd1 := MakeTestCommand([]string{}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	cmd2 := MakeTestCommand([]string{}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands = node.CollectCommands([]string{}, true)
	require.Len(t, commands, 2)

	commands = node.CollectCommands([]string{"test"}, true)
	assert.Len(t, commands, 1)

	commands = node.CollectCommands([]string{}, true)
	assert.Len(t, commands, 2)
}

func TestAddCommandDeeperLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	assert.Len(t, node.Children, 1)
	aNode := node.Children["a"]
	assert.Len(t, aNode.Children, 1)
	bNode := aNode.Children["b"]
	assert.Len(t, bNode.Children, 1)
	cNode := bNode.Children["c"]
	assert.Len(t, cNode.Children, 0)
	assert.Len(t, cNode.Commands, 1)

	cmd2 := MakeTestCommand([]string{"a", "b", "c"}, "test2")
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

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := MakeTestCommand([]string{"a", "b1", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)
	cmd3 := MakeTestCommand([]string{"a", "b2", "c"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)
	cmd4 := MakeTestCommand([]string{"a", "b", "c"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)
	cmd5 := MakeTestCommand([]string{"a", "b1", "c"}, "test5")
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

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := MakeTestCommand([]string{"a", "b", "c"}, "test2")
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

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := MakeTestCommand([]string{"a", "b1", "c"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)
	cmd3 := MakeTestCommand([]string{"a", "b2", "c"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)
	cmd4 := MakeTestCommand([]string{"a", "b", "c"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)
	cmd5 := MakeTestCommand([]string{"a", "b1", "c"}, "test5")
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

func TestInsertDuplicateCommand(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test", commands[0].Description().Name)
}

func TestRemoveNonexistentCommand(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)

	removedCommands := node.Remove([]string{"a", "b", "c", "test2"})
	assert.Len(t, removedCommands, 0)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 1)
	assert.Equal(t, "test", commands[0].Description().Name)
}

// TestInsertCommandWithSamePrefix tests that inserting a command with the same prefix as an existing command
// does not remove the existing command.
//
// Because we have aliases, we can't just remove the existing command when we insert a new one. For example,
// if we have a command "a b c" and we insert a command "a b c d", we can't just remove the existing command
// because it may be aliased to "a b c". This mirrors the behaviour on the CLI.
func TestInsertCommandWithSamePrefix(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"a", "b", "c"}, "test")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd2 := MakeTestCommand([]string{"a", "b", "c", "d"}, "test2")
	node.InsertCommand(cmd2.Description().Parents, cmd2)

	commands := node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 2)
	allNames := getNames(commands)
	assert.Contains(t, allNames, "test", "test2")

	// insert a node at a b c test foobar
	cmd3 := MakeTestCommand([]string{"a", "b", "c", "test", "foobar"}, "test3")
	node.InsertCommand(cmd3.Description().Parents, cmd3)

	commands = node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 3)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test", "test2", "test3")
}

func TestCollectCommandsNonRecursiveTwoLevels(t *testing.T) {
	node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
	require.NotNil(t, node)

	cmd1 := MakeTestCommand([]string{"a"}, "b")
	node.InsertCommand(cmd1.Description().Parents, cmd1)
	cmd4 := MakeTestCommand([]string{"a", "b", "c"}, "test4")
	node.InsertCommand(cmd4.Description().Parents, cmd4)

	commands := node.CollectCommands([]string{"a", "b"}, false)
	require.Len(t, commands, 1)
	allNames := getNames(commands)
	assert.Contains(t, allNames, "b")

	commands = node.CollectCommands([]string{"a", "b"}, true)
	require.Len(t, commands, 2)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "b", "test4")

	commands = node.CollectCommands([]string{"a", "b", "c"}, true)
	require.Len(t, commands, 1)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test4")

	commands = node.CollectCommands([]string{"a", "b", "c"}, false)
	require.Len(t, commands, 0)

	commands = node.CollectCommands([]string{"a", "b", "c", "test4"}, true)
	require.Len(t, commands, 1)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test4")

	commands = node.CollectCommands([]string{"a", "b", "c", "test4"}, false)
	require.Len(t, commands, 1)
	allNames = getNames(commands)
	assert.Contains(t, allNames, "test4")

}
