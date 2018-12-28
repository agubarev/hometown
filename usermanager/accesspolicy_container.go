package usermanager

// AccessPolicyObject represents an object with AccessPolicy
type AccessPolicyObject interface {
	AccessPolicy()
}

// AccessPolicyContainer is a registry for convenience
type AccessPolicyContainer struct {
	// TODO
}
