package accesspolicy

import "context"

// AccessResolver is responsible for resolving the final access rights
// when the rights are extended or inherited from a parent
type AccessResolver interface {
	Resolve(
		ctx context.Context,
		parentPolicy Policy,
		parentRight Right,
		ownRight Right,
	) (calculatedRight Right, err error)
}

type defaultResolver struct{}

func NewDefaultResolver() AccessResolver {
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
