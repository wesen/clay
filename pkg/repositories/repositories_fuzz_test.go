package repositories

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

type FuzzVerb int

const (
	// Add a command to the trie
	Add FuzzVerb = iota
	// Remove a command from the trie
	Remove
	// Collect all the commands in the trie
	Collect
	RemovePrefix
)

type FuzzInput struct {
	prefix []string
	name   string
	verb   FuzzVerb
}

func verbToString(verb FuzzVerb) string {
	switch verb {
	case Add:
		return "Add"
	case Remove:
		return "Remove"
	case Collect:
		return "Collect"
	case RemovePrefix:
		return "RemovePrefix"
	default:
		return "Unknown"
	}
}

func stringToVerb(str string) FuzzVerb {
	switch str {
	case "Add":
		return Add
	case "Remove":
		return Remove
	case "Collect":
		return Collect
	case "RemovePrefix":
		return RemovePrefix
	default:
		return -1
	}
}

func (f FuzzInput) String() string {
	return strings.Join(f.prefix, " ") + " " + f.name + " " + verbToString(f.verb)
}

func InputFromString(str string) FuzzInput {
	parts := strings.Split(str, " ")
	if len(parts) < 3 {
		return FuzzInput{
			[]string{},
			"",
			-1,
		}
	}
	verb := stringToVerb(parts[len(parts)-1])
	name := parts[len(parts)-2]
	prefix := parts[:len(parts)-2]
	return FuzzInput{prefix, name, verb}
}

func getFullNames(cmds_ []cmds.Command) []string {
	names := make([]string, len(cmds_))
	for i, c := range cmds_ {
		names[i] = getName(c)
	}
	return names
}

func getName(c cmds.Command) string {
	return strings.Join(c.Description().Parents, "/") + "/" + c.Description().Name
}

func inputsToString(inputs []FuzzInput) string {
	var strs []string
	for _, input := range inputs {
		strs = append(strs, input.String())
	}
	return strings.Join(strs, ";")
}

func inputsFromString(str string) []FuzzInput {
	var inputs []FuzzInput
	for _, s := range strings.Split(str, ";") {
		inputs = append(inputs, InputFromString(s))
	}
	return inputs
}

func FuzzTrieNode(f *testing.F) {
	inputs := []FuzzInput{
		{[]string{"a", "b", "c"}, "test", Add},
		{[]string{"a"}, "test", Collect},
		{[]string{"a", "b", "c"}, "test", Remove},
		{[]string{"a"}, "test", RemovePrefix},
	}
	f.Add(inputsToString(inputs))
	f.Fuzz(func(t *testing.T, inputStr string) {
		inputs := inputsFromString(inputStr)

		node := NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
		require.NotNil(t, node)

		for _, input := range inputs {
			switch input.verb {
			case Add:
				if input.name == "" {
					continue
				}
				for _, p := range input.prefix {
					if p == "" {
						continue
					}
				}
				cmd := MakeTestCommand(input.prefix, input.name)
				node.InsertCommand(cmd.Description().Parents, cmd)

				// make sure the node is here
				cmds := node.CollectCommands(input.prefix, true)
				namesBefore := getFullNames(cmds)
				names := getFullNames(cmds)
				cmdName := getName(cmd)
				assert.Contains(t, names, cmdName)

				// remove the node
				node.Remove(append(input.prefix, input.name))

				// make sure the node is not here
				cmds = node.CollectCommands(input.prefix, true)
				names = getFullNames(cmds)

				// manually check if names does contain cmdName
				for _, name := range names {
					if name == cmdName {
						t.Errorf("names: %v, namesBefore: %v, cmdName: %s\n", names, namesBefore, cmdName)
						t.Errorf("names contains cmdName")
					}
				}

				assert.NotContains(t, names, cmdName)

				// re-add the node
				node.InsertCommand(cmd.Description().Parents, cmd)
				cmds = node.CollectCommands(input.prefix, true)
				names = getFullNames(cmds)
				assert.Contains(t, names, cmdName)

			case Remove:
				if input.name == "" {
					continue
				}
				for _, p := range input.prefix {
					if p == "" {
						continue
					}
				}
				node.Remove(append(input.prefix, input.name))
				// make sure the node is gone
				cmds := node.CollectCommands(input.prefix, true)
				names := getFullNames(cmds)
				cmdName := getName(MakeTestCommand(input.prefix, input.name))
				assert.NotContains(t, names, cmdName)

				// if prefix > 1, make sure the parent does not contain the node
				if len(input.prefix) > 1 {
					parentPrefix := input.prefix[:len(input.prefix)-1]
					cmds = node.CollectCommands(parentPrefix, true)
					names = getFullNames(cmds)
					assert.NotContains(t, names, cmdName)
				}

			case Collect:
				commands := node.CollectCommands(input.prefix, true)
				// Check that all collected commands have the expected prefix
				for _, cmd := range commands {
					require.LessOrEqual(t, len(input.prefix), len(cmd.Description().Parents))
					parentPrefix := cmd.Description().Parents[:len(input.prefix)]
					assert.Equal(t, input.prefix, parentPrefix)
				}

			case RemovePrefix:
				node.Remove(input.prefix)

				after := node.CollectCommands(input.prefix, true)
				afterNames := getFullNames(after)

				assert.Empty(t, afterNames)

				// make sure the node is gone
			}
		}
	})
}
