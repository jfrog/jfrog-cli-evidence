package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeFieldsBuilder_BaseFieldsOnly(t *testing.T) {
	result := NewNodeFieldsBuilder("fieldA", "fieldB").Build()
	assert.Equal(t, "fieldA fieldB", result)
}

func TestNodeFieldsBuilder_EmptyBuilder(t *testing.T) {
	result := NewNodeFieldsBuilder().Build()
	assert.Equal(t, "", result)
}

func TestNodeFieldsBuilder_WithIf_True(t *testing.T) {
	result := NewNodeFieldsBuilder("base").
		WithIf(true, "optional").
		Build()
	assert.Equal(t, "base optional", result)
}

func TestNodeFieldsBuilder_WithIf_False(t *testing.T) {
	result := NewNodeFieldsBuilder("base").
		WithIf(false, "optional").
		Build()
	assert.Equal(t, "base", result)
}

func TestNodeFieldsBuilder_Chaining(t *testing.T) {
	result := NewNodeFieldsBuilder("a", "b").
		WithIf(true, "c").
		WithIf(false, "d").
		WithIf(true, "e").
		Build()
	assert.Equal(t, "a b c e", result)
}

func TestNodeFieldsBuilder_WithAttachmentsFragment(t *testing.T) {
	result := NewNodeFieldsBuilder(FieldDownloadPath, FieldSubjectSha256).
		WithIf(true, AttachmentsFragment).
		Build()
	assert.Contains(t, result, FieldDownloadPath)
	assert.Contains(t, result, FieldSubjectSha256)
	assert.Contains(t, result, AttachmentsFragment)
}

func TestNodeFieldsBuilder_WithoutAttachmentsFragment(t *testing.T) {
	result := NewNodeFieldsBuilder(FieldDownloadPath, FieldSubjectSha256).
		WithIf(false, AttachmentsFragment).
		Build()
	assert.NotContains(t, result, "attachments")
}

func TestBuildQuery_ReplacesPlaceholder(t *testing.T) {
	template := `{"query":"{ node { ` + NodeFieldsPlaceholder + ` } }"}`
	result := BuildQuery(template, "fieldA fieldB")
	assert.Equal(t, `{"query":"{ node { fieldA fieldB } }"}`, result)
}

func TestBuildQuery_ReplacesMultiplePlaceholders(t *testing.T) {
	template := "first { " + NodeFieldsPlaceholder + " } second { " + NodeFieldsPlaceholder + " }"
	result := BuildQuery(template, "a b")
	assert.Equal(t, "first { a b } second { a b }", result)
}

func TestBuildQuery_NoPlaceholder(t *testing.T) {
	template := `{"query":"{ node { fixed } }"}`
	result := BuildQuery(template, "ignored")
	assert.Equal(t, template, result)
}

func TestAttachmentsFragmentValue(t *testing.T) {
	assert.Equal(t, "attachments { name sha256 type downloadPath }", AttachmentsFragment)
}

func TestIsAttachmentsFieldNotFound_NilError(t *testing.T) {
	assert.False(t, IsAttachmentsFieldNotFound(nil))
}

func TestIsAttachmentsFieldNotFound_ExactOneModelError(t *testing.T) {
	err := fmt.Errorf(`Cannot query field "attachments" on type "Evidence".`)
	assert.True(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_MatchesGraphQLValidationError(t *testing.T) {
	err := fmt.Errorf(`Cannot query field "attachments" on type "EvidenceNode"`)
	assert.True(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_MatchesWrappedError(t *testing.T) {
	err := fmt.Errorf(`graphql error: Cannot query field "attachments" on type "EvidenceNode". Did you mean "attachment"?`)
	assert.True(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_UnrelatedFieldError(t *testing.T) {
	err := fmt.Errorf(`Cannot query field "someOtherField" on type "EvidenceQueries"`)
	assert.False(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_UnrelatedError(t *testing.T) {
	err := fmt.Errorf("connection refused")
	assert.False(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_PartialMatchCannotQueryOnly(t *testing.T) {
	err := fmt.Errorf(`Cannot query field "publicKey" on type "EvidenceNode"`)
	assert.False(t, IsAttachmentsFieldNotFound(err))
}

func TestIsAttachmentsFieldNotFound_PartialMatchAttachmentsOnly(t *testing.T) {
	err := fmt.Errorf("failed to load attachments from database")
	assert.False(t, IsAttachmentsFieldNotFound(err))
}
