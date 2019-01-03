package usermanager

// AccessPolicyObject represents an object with AccessPolicy
type AccessPolicyObject interface {
	AccessPolicy()
}

// AccessPolicyContainer is a registry for convenience
// TODO: container should be responsible for a relationship between policies and actual objects (allow multiple objects per policy)
type AccessPolicyContainer struct {
}
