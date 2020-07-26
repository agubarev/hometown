package user

// Container represents a collection of user objects to enable convenient
type Container []User

func (c Container) Validate() (err error) {
	for _, u := range c {
		if err = u.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Filter filters container by a given predicate function
func (c Container) Filter(fn func(User) bool) (result Container) {
	result = make(Container, 0)

	// looping over the current user list and filtering by a given function
	for i := range c {
		if fn(c[i]) {
			result = append(result, c[i])
		}
	}

	return result
}
