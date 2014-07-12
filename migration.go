// Holds metadata about a migration.

package gomigrate

// Migration statuses.
const (
	inactive = iota
	active
)

// Holds configuration information for a given migration.
type Migration struct {
	DownPath string
	Name     string
	Num      uint64
	Status   int
	UpPath   string
}

// Performs a basic validation of a migration.
func (m *Migration) valid() bool {
	if m.Num != 0 && m.Name != "" && m.UpPath != "" && m.DownPath != "" {
		return true
	}
	return false
}
