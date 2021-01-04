package accesspolicy

import "context"

// Resolver is responsible for resolving the final access rights
// when the rights are extended or inherited from a parent
type Resolver interface {
	// Resolve calculates the final access right value of a policy
	// which extends or inherits from a parent, because sometimes a certain right
	// must be overridden while still preserving extended some values
	Resolve(ctx context.Context, parentPolicy Policy, parentRight Right, ownRight Right) (calculatedRight Right, err error)
}

type defaultResolver struct{}

func NewDefaultResolver() Resolver {
	return &defaultResolver{}
}

func (d *defaultResolver) Resolve(
	ctx context.Context,
	parentPolicy Policy,
	parentRight Right,
	ownRight Right,
) (calculatedRight Right, err error) {
	panic("implement me")
}
