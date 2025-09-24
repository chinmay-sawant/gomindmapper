// Package ent provides mock types for testing external module call resolution
package ent

// FormDatastoreClient represents an external interface that should be analyzed
type FormDatastoreClient interface {
	GetFormId() (string, error)
	SaveForm(id string, data string) error
	CreateNew() error
	DeleteForm(id string) error
}
