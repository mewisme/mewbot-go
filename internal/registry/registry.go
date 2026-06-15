package registry

import (
	"strings"

	"github.com/mewisme/mewbot-go/internal/command"
)

// Registry holds registered commands and lookup maps for slash and prefix use.
type Registry struct {
	commands  map[string]command.Command
	prefixMap map[string]command.Command
	order     []string
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{
		commands:  map[string]command.Command{},
		prefixMap: map[string]command.Command{},
	}
}

// Register adds a command and wires up its prefix and alias lookups.
func (r *Registry) Register(c command.Command) {
	name := strings.ToLower(c.Name())
	if _, exists := r.commands[name]; !exists {
		r.order = append(r.order, name)
	}
	r.commands[name] = c

	if _, ok := r.prefixMap[name]; !ok {
		r.prefixMap[name] = c
	}
	if p := c.Prefix(); p != "" {
		r.prefixMap[strings.ToLower(p)] = c
	}
	for _, alias := range c.Aliases() {
		al := strings.ToLower(alias)
		if _, ok := r.prefixMap[al]; !ok {
			r.prefixMap[al] = c
		}
	}
}

// GetSlash returns the command registered under the given slash name.
func (r *Registry) GetSlash(name string) (command.Command, bool) {
	c, ok := r.commands[strings.ToLower(name)]
	return c, ok
}

// GetPrefix returns the command registered under a prefix trigger or alias.
func (r *Registry) GetPrefix(name string) (command.Command, bool) {
	c, ok := r.prefixMap[strings.ToLower(name)]
	return c, ok
}

// All returns the registered commands in registration order.
func (r *Registry) All() []command.Command {
	out := make([]command.Command, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.commands[name])
	}
	return out
}

// List returns flattened command info for help rendering.
func (r *Registry) List() []command.Info {
	cmds := r.All()
	out := make([]command.Info, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, command.Info{
			Name:        c.Name(),
			Description: c.Description(),
			Prefix:      c.Prefix(),
			Aliases:     c.Aliases(),
			CooldownSec: int(c.Cooldown().Seconds()),
			Version:     c.Version(),
			Subcommands: c.Subcommands(),
		})
	}
	return out
}
