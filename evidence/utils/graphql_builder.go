package utils

import "strings"

const (
	NodeFieldsPlaceholder = "{{NODE_FIELDS}}"

	// Shared GraphQL node field fragments.
	FieldDownloadPath            = "downloadPath"
	FieldPredicateType           = "predicateType"
	FieldPredicateSlug           = "predicateSlug"
	FieldPredicate               = "predicate"
	FieldVerified                = "verified"
	FieldCreatedBy               = "createdBy"
	FieldCreatedAt               = "createdAt"
	FieldSubjectSha256           = "subject { sha256 }"
	FieldSigningKeyAlias         = "signingKey { alias }"
	FieldSigningKeyWithPublicKey = "signingKey {alias, publicKey}"

	// Optional GraphQL fragments.
	AttachmentsFragment = "attachments { name sha256 type downloadPath }"
)

// NodeFieldsBuilder composes GraphQL node field lists dynamically,
// avoiding combinatorial explosion of query constants when optional
// fields (attachments, predicate, publicKey, etc.) are involved.
type NodeFieldsBuilder struct {
	fields []string
}

// NewNodeFieldsBuilder creates a builder pre-populated with the given base fields.
func NewNodeFieldsBuilder(baseFields ...string) *NodeFieldsBuilder {
	return &NodeFieldsBuilder{fields: append([]string{}, baseFields...)}
}

// WithIf appends field to the list only when condition is true.
func (b *NodeFieldsBuilder) WithIf(condition bool, field string) *NodeFieldsBuilder {
	if condition {
		b.fields = append(b.fields, field)
	}
	return b
}

// Build returns the space-joined field list ready for insertion into a query template.
func (b *NodeFieldsBuilder) Build() string {
	return strings.Join(b.fields, " ")
}

// BuildQuery replaces every occurrence of NodeFieldsPlaceholder in template
// with the provided nodeFields string.
func BuildQuery(template, nodeFields string) string {
	return strings.ReplaceAll(template, NodeFieldsPlaceholder, nodeFields)
}
