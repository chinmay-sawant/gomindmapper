package example

import (
	"context"

	"github.com/chinmay-sawant/gomindmapper/example/ent"
)

// Service demonstrates the external library call resolution issue
type Service struct {
	FormDatastore ent.FormDatastoreClient
}

// GeneratePDFDocument demonstrates a function that calls external library methods
func (svc Service) GeneratePDFDocument(ctx context.Context, request string) error {
	// This should be resolved to ent.FormDatastoreClient.GetFormId
	formId, err := svc.FormDatastore.GetFormId()
	if err != nil {
		return err
	}

	// This should also be resolved to ent.FormDatastoreClient.SaveForm
	err = svc.FormDatastore.SaveForm(formId, request)
	if err != nil {
		return err
	}

	return nil
}

// AnotherFunction demonstrates direct interface calls
func AnotherFunction(client ent.FormDatastoreClient) {
	// This should be resolved to ent.FormDatastoreClient.CreateNew
	client.CreateNew()
}
