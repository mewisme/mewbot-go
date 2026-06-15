package appconfig

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mewisme/mewbot-go/internal/permissions"
	"github.com/mewisme/mewbot-go/internal/store"
)

// ConfigFile is the data-dir filename backing the perm tree.
const ConfigFile = "config.json"

// MaxDepth is the deepest command path the perm tree accepts.
const MaxDepth = 4

// node is a single perm-tree entry. The schema is uniform at every depth:
//
//	all      - user IDs allowed at this node AND all descendant nodes (inherits)
//	users    - user IDs allowed at this exact node only (no inheritance)
//	children - child nodes
type node struct {
	All      []string         `json:"all"`
	Users    []string         `json:"users"`
	Children map[string]*node `json:"children"`
}

func newNode() *node {
	return &node{All: []string{}, Users: []string{}, Children: map[string]*node{}}
}

// file is the on-disk JSON shape.
type file struct {
	Perm map[string]*node `json:"perm"`
}

// Manager holds the perm tree in memory and persists changes to config.json.
type Manager struct {
	mu   sync.RWMutex
	perm map[string]*node
}

// Load reads data/config.json into a Manager, creating an empty tree if absent.
func Load() (*Manager, error) {
	m := &Manager{perm: map[string]*node{}}

	raw, err := store.Load(ConfigFile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return m, nil
	}

	var f file
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ConfigFile, err)
	}
	if f.Perm != nil {
		m.perm = f.Perm
	}
	normalizeMap(m.perm)
	return m, nil
}

// normalizeMap ensures every node has non-nil slices/maps after unmarshalling.
func normalizeMap(nodes map[string]*node) {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.All == nil {
			n.All = []string{}
		}
		if n.Users == nil {
			n.Users = []string{}
		}
		if n.Children == nil {
			n.Children = map[string]*node{}
		}
		normalizeMap(n.Children)
	}
}

// ValidatePath checks that a path is non-empty and within the depth limit.
func ValidatePath(path []string) error {
	if len(path) == 0 {
		return fmt.Errorf("path must have at least one segment")
	}
	if len(path) > MaxDepth {
		return fmt.Errorf("path depth %d exceeds max %d", len(path), MaxDepth)
	}
	for _, seg := range path {
		if strings.TrimSpace(seg) == "" {
			return fmt.Errorf("path segments must not be empty")
		}
	}
	return nil
}

// SplitPath turns a dotted path (e.g. "wallet.set") into segments.
func SplitPath(dotted string) []string {
	parts := strings.Split(dotted, ".")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

// find returns the node at path without creating intermediates.
func (m *Manager) find(path []string) *node {
	if len(path) == 0 {
		return nil
	}
	cur := m.perm[path[0]]
	for i := 1; i < len(path) && cur != nil; i++ {
		cur = cur.Children[path[i]]
	}
	return cur
}

// ensure returns the node at path, creating intermediates as needed.
func (m *Manager) ensure(path []string) *node {
	cur := m.perm[path[0]]
	if cur == nil {
		cur = newNode()
		m.perm[path[0]] = cur
	}
	for i := 1; i < len(path); i++ {
		next := cur.Children[path[i]]
		if next == nil {
			next = newNode()
			cur.Children[path[i]] = next
		}
		cur = next
	}
	return cur
}

// Bucket selects which list within a node a perm operation targets.
type Bucket int

const (
	// BucketUsers grants access at the exact node only (no inheritance).
	BucketUsers Bucket = iota
	// BucketAll grants access at the node and all descendant nodes.
	BucketAll
)

// AllSegment is the reserved dotted-path segment that targets the "all" bucket.
const AllSegment = "all"

// RoleMember is a grant token: any guild member passes.
const RoleMember = "member"

// RoleAdmin is a grant token: guild admin or owner passes.
const RoleAdmin = "admin"

// IsGrantToken reports whether s is a role grant token (member or admin).
func IsGrantToken(s string) bool {
	return s == RoleMember || s == RoleAdmin
}

func entryMatches(entry, userID string, level permissions.Level) bool {
	switch entry {
	case RoleMember:
		return level >= permissions.LevelMember
	case RoleAdmin:
		return level >= permissions.LevelAdmin
	default:
		return entry == userID
	}
}

func (n *node) list(b Bucket) *[]string {
	if b == BucketAll {
		return &n.All
	}
	return &n.Users
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// Allowed reports whether userID may invoke the command at path. A user is
// granted when an entry in the exact node's "users" bucket matches, or an
// entry in the "all" bucket of that node or any ancestor matches. Entries
// may be user IDs or role tokens ("member", "admin").
func (m *Manager) Allowed(path []string, userID string, level permissions.Level) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := 1; i <= len(path); i++ {
		n := m.find(path[:i])
		if n == nil {
			continue
		}
		for _, entry := range n.All {
			if entryMatches(entry, userID, level) {
				return true
			}
		}
		if i == len(path) {
			for _, entry := range n.Users {
				if entryMatches(entry, userID, level) {
					return true
				}
			}
		}
	}
	return false
}

// BucketAllows reports whether userID is allowed by entries in the given
// bucket at the exact node for path.
func (m *Manager) BucketAllows(path []string, bucket Bucket, userID string, level permissions.Level) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.find(path)
	if n == nil {
		return false
	}
	for _, entry := range *n.list(bucket) {
		if entryMatches(entry, userID, level) {
			return true
		}
	}
	return false
}

// InBucket reports whether entry is listed in the bucket at the exact node
// (literal membership, not role resolution).
func (m *Manager) InBucket(path []string, bucket Bucket, entry string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.find(path)
	if n == nil {
		return false
	}
	return contains(*n.list(bucket), entry)
}

// Add inserts grants (user IDs or role tokens) into the bucket at path and
// persists. Returns the grants newly added.
func (m *Manager) Add(path []string, bucket Bucket, grants []string) ([]string, error) {
	m.mu.Lock()
	n := m.ensure(path)
	list := n.list(bucket)
	existing := map[string]struct{}{}
	for _, id := range *list {
		existing[id] = struct{}{}
	}
	var added []string
	for _, g := range grants {
		if _, ok := existing[g]; ok {
			continue
		}
		existing[g] = struct{}{}
		*list = append(*list, g)
		added = append(added, g)
	}
	m.mu.Unlock()

	if len(added) == 0 {
		return added, nil
	}
	return added, m.save()
}

// Remove deletes grants from the bucket at path and persists.
func (m *Manager) Remove(path []string, bucket Bucket, grants []string) ([]string, error) {
	m.mu.Lock()
	n := m.find(path)
	if n == nil {
		m.mu.Unlock()
		return nil, nil
	}
	list := n.list(bucket)
	drop := map[string]struct{}{}
	for _, g := range grants {
		drop[g] = struct{}{}
	}
	kept := (*list)[:0:0]
	var removed []string
	for _, id := range *list {
		if _, ok := drop[id]; ok {
			removed = append(removed, id)
			continue
		}
		kept = append(kept, id)
	}
	*list = kept
	m.mu.Unlock()

	if len(removed) == 0 {
		return removed, nil
	}
	return removed, m.save()
}

// List returns a copy of grants in the bucket at the exact node for path.
func (m *Manager) List(path []string, bucket Bucket) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.find(path)
	if n == nil {
		return nil
	}
	src := *n.list(bucket)
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func (m *Manager) save() error {
	m.mu.RLock()
	f := file{Perm: m.perm}
	b, err := json.MarshalIndent(f, "", "  ")
	m.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("encode %s: %w", ConfigFile, err)
	}
	return store.Save(ConfigFile, string(b))
}
