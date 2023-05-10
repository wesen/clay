package repositories

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/rs/zerolog/log"
)

// A repository is a collection of commands and aliases, that can optionally be reloaded
// through a watcher (and for which you can register callbacks, for example to update a potential
// cobra command or REST route).

type UpdateCallback func(cmd cmds.Command) error
type RemoveCallback func(cmd cmds.Command) error

type TrieNode struct {
	Children map[string]*TrieNode
	Commands []cmds.Command
}

// NewTrieNode creates a new trie node.
func NewTrieNode(commands []cmds.Command, aliases []*alias.CommandAlias) *TrieNode {
	return &TrieNode{
		Children: make(map[string]*TrieNode),
		Commands: commands,
	}
}

// Remove removes a command from the trie.
func (t *TrieNode) Remove(prefix []string) []cmds.Command {
	if len(prefix) == 0 {
		commands := t.CollectCommands(prefix, true)
		t.Commands = make([]cmds.Command, 0)
		t.Children = make(map[string]*TrieNode)

		return commands
	}

	removedCommands := make([]cmds.Command, 0)

	// try to get parent node
	path := prefix[:len(prefix)-1]
	parentNode := t.findNode(path, false)
	name := prefix[len(prefix)-1]
	if parentNode == nil {
		log.Debug().Msgf("parent node not found for %s", name)
		return []cmds.Command{}
	}

	childNode, ok := parentNode.Children[name]
	if ok {

		// remove the node
		commands := childNode.CollectCommands([]string{}, true)
		removedCommands = append(removedCommands, commands...)

		delete(parentNode.Children, name)
	}
	// check if this is an actual command or alias
	for i, c := range parentNode.Commands {
		if c.Description().Name == name {
			removedCommands = append(removedCommands, c)
			parentNode.Commands = append(parentNode.Commands[:i], parentNode.Commands[i+1:]...)
		}
	}

	return removedCommands
}

// InsertCommand inserts a command in the trie, replacing it if it already exists.
func (t *TrieNode) InsertCommand(prefix []string, command cmds.Command) {
	node := t.findNode(prefix, true)

	// check if the command is already in the trie
	for i, c := range node.Commands {
		if c.Description().Name == command.Description().Name {
			node.Commands[i] = command
			return
		}
	}

	node.Commands = append(node.Commands, command)
}

// findNode finds the node corresponding to the given prefix, creating it if it doesn't exist.
func (t *TrieNode) findNode(prefix []string, createNewNodes bool) *TrieNode {
	node := t
	for _, p := range prefix {
		if _, ok := node.Children[p]; !ok {
			if !createNewNodes {
				log.Debug().Msgf("node %s not found", p)
				return nil
			}
			node.Children[p] = NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{})
		}
		node = node.Children[p]
	}
	return node
}

func (t *TrieNode) FindCommand(path []string) (cmds.Command, bool) {
	if len(path) == 0 {
		return nil, false
	}
	parentPath := path[:len(path)-1]
	commandName := path[len(path)-1]
	node := t.findNode(parentPath, false)
	if node == nil {
		return nil, false
	}

	for _, c := range node.Commands {
		if c.Description().Name == commandName {
			return c, true
		}
	}

	return nil, false
}

// CollectCommands collects all commands and aliases under the given prefix.
func (t *TrieNode) CollectCommands(prefix []string, recurse bool) []cmds.Command {
	ret := make([]cmds.Command, 0)

	// Check if the prefix identifies a single command.
	if len(prefix) > 0 {
		// try to get parent node
		path := prefix[:len(prefix)-1]
		parentNode := t.findNode(path, false)
		name := prefix[len(prefix)-1]
		if parentNode != nil {
			for _, c := range parentNode.Commands {
				if c.Description().Name == name {
					ret = append(ret, c)
					break
				}
			}
		}

		if !recurse {
			return ret
		}
	}

	node := t.findNode(prefix, false)
	if node == nil {
		return ret
	}

	if !recurse {
		return node.Commands
	}

	// recurse into node to collect all commands and aliases
	for _, child := range node.Children {
		c := child.CollectCommands([]string{}, true)
		ret = append(ret, c...)
	}

	// add commands and aliases from current node
	ret = append(ret, node.Commands...)

	return ret
}

type Repository struct {
	// The root of the repository.
	Root           *TrieNode
	Directories    []string
	updateCallback UpdateCallback
	removeCallback RemoveCallback
}

type RepositoryOption func(*Repository)

func WithDirectories(directories []string) RepositoryOption {
	return func(r *Repository) {
		r.Directories = directories
	}
}

func WithDirectory(directory string) RepositoryOption {
	return func(r *Repository) {
		r.Directories = append(r.Directories, directory)
	}
}

func WithUpdateCallback(callback UpdateCallback) RepositoryOption {
	return func(r *Repository) {
		r.updateCallback = callback
	}
}

func WithRemoveCallback(callback RemoveCallback) RepositoryOption {
	return func(r *Repository) {
		r.removeCallback = callback
	}
}

// NewRepository creates a new repository.
func NewRepository(options ...RepositoryOption) *Repository {
	ret := &Repository{
		Root: NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{}),
	}
	for _, opt := range options {
		opt(ret)
	}
	return ret
}

func (r *Repository) Add(commands ...cmds.Command) {
	aliases := []*alias.CommandAlias{}

	for _, command := range commands {
		_, isAlias := command.(*alias.CommandAlias)
		if isAlias {
			aliases = append(aliases, command.(*alias.CommandAlias))
			continue
		}

		prefix := command.Description().Parents
		r.Root.InsertCommand(prefix, command)
		if r.updateCallback != nil {
			err := r.updateCallback(command)
			if err != nil {
				log.Warn().Err(err).Msg("error while updating command")
			}
		}
	}

	for _, alias_ := range aliases {
		prefix := alias_.Parents
		aliasedCommand, ok := r.Root.FindCommand(prefix)
		if !ok {
			log.Warn().Msgf("alias_ %s for %s not found", alias_.Description().Name, alias_.AliasFor)
			continue
		}
		alias_.AliasedCommand = aliasedCommand

		r.Root.InsertCommand(prefix, alias_)
		if r.updateCallback != nil {
			err := r.updateCallback(alias_)
			if err != nil {
				log.Warn().Err(err).Msg("error while updating command")
			}
		}
	}
}

func (r *Repository) Remove(prefixes ...[]string) {
	for _, prefix := range prefixes {
		removedCommands := r.Root.Remove(prefix)
		for _, command := range removedCommands {
			if r.removeCallback != nil {
				err := r.removeCallback(command)
				if err != nil {
					log.Warn().Err(err).Msg("error while removing command")
				}
			}
		}
	}
}
