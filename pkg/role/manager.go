package role

// Manager is a role manager contract interface
type Manager interface {
	List() []*Role
	Create(name string, parent *Role) (*Role, error)
	Get(name string) *Role
	Update(r *Role) error
	Delete(r *Role) error
}

type manager struct {
	store Store
}

// NewDefaultManager initializing a default Role manager
func NewDefaultManager(s Store) Manager {

}
