package kdbx

import (
	"fmt"
	"os"
	"time"

	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

// DB wraps a KeePass database with its file path and credentials.
type DB struct {
	Path     string
	Password string
	db       *gokeepasslib.Database
}

// Entry represents a single stored manifest/secret in KeePass.
type Entry struct {
	Title     string
	Group     string // namespace or logical group
	Content   string // raw YAML/JSON
	Kind      string // e.g. Secret, ConfigMap
	Name      string
	Namespace string
	Modified  time.Time
}

// CreateDB initialises a new KeePass database file.
func CreateDB(path, password string) error {
	root := gokeepasslib.NewGroup()
	root.Name = "KubeKee"

	db := gokeepasslib.NewDatabase(
		gokeepasslib.WithDatabaseKDBXVersion4(),
	)
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	db.Content.Root = &gokeepasslib.RootData{
		Groups: []gokeepasslib.Group{root},
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating db file: %w", err)
	}
	defer f.Close()

	if err := db.LockProtectedEntries(); err != nil {
		return fmt.Errorf("locking entries: %w", err)
	}

	enc := gokeepasslib.NewEncoder(f)
	if err := enc.Encode(db); err != nil {
		return fmt.Errorf("encoding db: %w", err)
	}
	return nil
}

// OpenDB opens an existing KeePass database.
func OpenDB(path, password string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening db file: %w", err)
	}
	defer f.Close()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)

	dec := gokeepasslib.NewDecoder(f)
	if err := dec.Decode(db); err != nil {
		return nil, fmt.Errorf("decoding db: %w", err)
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return nil, fmt.Errorf("unlocking entries: %w", err)
	}

	return &DB{Path: path, Password: password, db: db}, nil
}

// Save writes the database back to disk.
func (d *DB) Save() error {
	f, err := os.Create(d.Path)
	if err != nil {
		return fmt.Errorf("creating db file: %w", err)
	}
	defer f.Close()

	if err := d.db.LockProtectedEntries(); err != nil {
		return fmt.Errorf("locking entries: %w", err)
	}

	enc := gokeepasslib.NewEncoder(f)
	if err := enc.Encode(d.db); err != nil {
		return fmt.Errorf("encoding db: %w", err)
	}
	return nil
}

// findOrCreateGroup finds or creates a group by name under root.
func (d *DB) findOrCreateGroup(name string) *gokeepasslib.Group {
	root := &d.db.Content.Root.Groups[0]
	if name == "" || name == root.Name {
		return root
	}
	for i := range root.Groups {
		if root.Groups[i].Name == name {
			return &root.Groups[i]
		}
	}
	g := gokeepasslib.NewGroup()
	g.Name = name
	root.Groups = append(root.Groups, g)
	return &root.Groups[len(root.Groups)-1]
}

func mkValue(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{
		Key:   key,
		Value: gokeepasslib.V{Content: value},
	}
}

func mkProtectedValue(key, value string) gokeepasslib.ValueData {
	return gokeepasslib.ValueData{
		Key: key,
		Value: gokeepasslib.V{
			Content:   value,
			Protected: w.NewBoolWrapper(true),
		},
	}
}

// AddEntry adds a manifest entry to the database.
func (d *DB) AddEntry(e Entry) error {
	group := d.findOrCreateGroup(e.Group)

	// Check for duplicates
	for _, existing := range group.Entries {
		if getVal(existing, "Title") == e.Title {
			return fmt.Errorf("entry %q already exists in group %q, use edit to update", e.Title, e.Group)
		}
	}

	entry := gokeepasslib.NewEntry()
	entry.Values = []gokeepasslib.ValueData{
		mkValue("Title", e.Title),
		mkValue("UserName", e.Kind),
		mkProtectedValue("Notes", e.Content),
		mkValue("URL", fmt.Sprintf("%s/%s", e.Namespace, e.Name)),
	}

	group.Entries = append(group.Entries, entry)
	return nil
}

// UpdateEntry updates an existing entry's content.
func (d *DB) UpdateEntry(title, group, newContent string) error {
	g := d.findOrCreateGroup(group)
	for i := range g.Entries {
		if getVal(g.Entries[i], "Title") == title {
			for j := range g.Entries[i].Values {
				if g.Entries[i].Values[j].Key == "Notes" {
					g.Entries[i].Values[j].Value.Content = newContent
					return nil
				}
			}
		}
	}
	return fmt.Errorf("entry %q not found in group %q", title, group)
}

// GetEntry retrieves an entry by title and optional group.
func (d *DB) GetEntry(title, group string) (*Entry, error) {
	groups := d.searchGroups(group)
	for _, g := range groups {
		for _, e := range g.Entries {
			if getVal(e, "Title") == title {
				return entryFromKeePass(e, g.Name), nil
			}
		}
	}
	return nil, fmt.Errorf("entry %q not found", title)
}

// ListEntries returns all entries, optionally filtered by group.
func (d *DB) ListEntries(group string) []Entry {
	var result []Entry
	groups := d.searchGroups(group)
	for _, g := range groups {
		for _, e := range g.Entries {
			result = append(result, *entryFromKeePass(e, g.Name))
		}
	}
	return result
}

func (d *DB) searchGroups(group string) []gokeepasslib.Group {
	root := d.db.Content.Root.Groups[0]
	if group != "" {
		for _, g := range root.Groups {
			if g.Name == group {
				return []gokeepasslib.Group{g}
			}
		}
		if root.Name == group {
			return []gokeepasslib.Group{root}
		}
		return nil
	}
	// Return root + all subgroups
	all := []gokeepasslib.Group{root}
	all = append(all, root.Groups...)
	return all
}

func getVal(e gokeepasslib.Entry, key string) string {
	for _, v := range e.Values {
		if v.Key == key {
			return v.Value.Content
		}
	}
	return ""
}

func entryFromKeePass(e gokeepasslib.Entry, groupName string) *Entry {
	return &Entry{
		Title:   getVal(e, "Title"),
		Group:   groupName,
		Content: getVal(e, "Notes"),
		Kind:    getVal(e, "UserName"),
	}
}
