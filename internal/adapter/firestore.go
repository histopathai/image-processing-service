package adapter

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FirestoreAdapter struct {
	client     *firestore.Client
	collection string
}

func NewFirestoreAdapter(client *firestore.Client, collection string) *FirestoreAdapter {
	return &FirestoreAdapter{
		client:     client,
		collection: collection,
	}
}

func (f *FirestoreAdapter) Create(ctx context.Context, data map[string]interface{}) (string, error) {
	docRef := f.client.Collection(f.collection).NewDoc()
	_, err := docRef.Set(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}
	return docRef.ID, nil
}

func (f *FirestoreAdapter) Read(ctx context.Context, id string) (map[string]interface{}, error) {
	docRef := f.client.Collection(f.collection).Doc(id)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read document: %w", err)
	}
	return doc.Data(), nil
}

func (f *FirestoreAdapter) Update(ctx context.Context, id string, data map[string]interface{}) error {
	docRef := f.client.Collection(f.collection).Doc(id)
	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	return nil
}

func (f *FirestoreAdapter) Delete(ctx context.Context, id string) error {
	docRef := f.client.Collection(f.collection).Doc(id)
	_, err := docRef.Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("document not found: %s", id)
		}
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

func (f *FirestoreAdapter) List(ctx context.Context, filter map[string]interface{}) ([]map[string]interface{}, error) {
	query := f.client.Collection(f.collection).Query
	for key, value := range filter {
		query = query.Where(key, "==", value)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var results []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("failed to list documents: %w", err)
		}
		results = append(results, doc.Data())
	}

	return results, nil
}
