// Holds metadata about a migration.

package gomigrate

// Migration statuses.
const (
	Inactive = iota
	Active
)

// Holds configuration information for a given migration.
type Migration struct {
	DownPath string
	Name     string
	Status   int
	UpPath   string

	// The file system identifier, not the database id.
	Id uint64
}

// Performs a basic validation of a migration.
func (m *Migration) valid() bool {
	if m.Id != 0 && m.Name != "" && m.UpPath != "" && m.DownPath != "" {
		return true
	}
	return false
}
