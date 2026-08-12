package model

// EntitySubject holds the identity and Artifactory scope for an entity evidence subject.
// Exactly one of EntityRepo, ProjectKey, and ApplicationKey may be set (or none).
type EntitySubject struct {
	EntityType     string
	EntityID       string
	EntityRepo     string
	ProjectKey     string
	ApplicationKey string
}
