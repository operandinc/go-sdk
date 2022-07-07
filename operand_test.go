package operand

import (
	"context"
	"os"
	"testing"
)

// TestOperand is a basic usage example of the Operand Go SDK + API.
func TestOperand(t *testing.T) {
	key, ok := os.LookupEnv("OPERAND_API_KEY")
	if !ok {
		t.Fatal("OPERAND_API_KEY not set")
	}

	// Create a new context to use for the tests.
	ctx := context.Background()

	// Create a new Operand client.
	client := NewClient(key)

	// Make a new collection, i.e. folder to keep test data.
	collection, err := client.CreateObject(ctx, CreateObjectArgs{
		Type:     ObjectTypeCollection,
		Metadata: CollectionMetadata{},
		Label:    AsRef("go-sdk-tests"),
	})
	if err != nil {
		t.Fatal(err)
	} else if err := collection.Wait(ctx, client); err != nil {
		t.Fatal(err)
	}

	// Make sure the collection indexed properly.
	if collection.IndexingStatus != IndexingStatusReady {
		t.Fatalf(
			"expected collection indexing status to be ready, got %s",
			collection.IndexingStatus,
		)
	}

	// Index a few text documents.
	documents := []string{
		"Operand makes knowledge come alive.",
		"We're super excited about our new Go SDK, we love Go!",
		"Rust is also pretty great, but it can definetly be a little bit more complex.",
	}
	var objects []*Object
	for _, d := range documents {
		obj, err := client.CreateObject(ctx, CreateObjectArgs{
			ParentID: AsRef(collection.ID),
			Type:     ObjectTypeText,
			Metadata: TextMetadata{
				Text: d,
			},
		})
		if err != nil {
			t.Fatal(err)
		} else if err := obj.Wait(ctx, client); err != nil {
			t.Fatal(err)
		}

		// Make sure the object indexed properly.
		if obj.IndexingStatus != IndexingStatusReady {
			t.Fatalf("expected object indexing status to be ready, got %s", obj.IndexingStatus)
		}

		objects = append(objects, obj)
	}

	// Do a few queries and make sure we're getting the right results.
	queries := []struct {
		query       string
		expectIndex int
	}{
		{
			query:       "what's operand?",
			expectIndex: 0,
		},
		{
			query:       "difficult programming language",
			expectIndex: 2,
		},
		{
			query:       "Go",
			expectIndex: 1,
		},
	}

	for _, q := range queries {
		resp, err := client.SearchContents(ctx, SearchContentsArgs{
			ParentIDs: []string{collection.ID},
			Query:     q.query,
			Max:       3,
		})
		if err != nil {
			t.Fatal(err)
		} else if len(resp.Contents) != 3 {
			t.Fatalf("expected 3 results, got %d", len(resp.Contents))
		} else if resp.Objects[resp.Contents[0].ObjectID].ID != objects[q.expectIndex].ID {
			t.Fatalf("expected top content result of %s, got %s", documents[q.expectIndex], resp.Contents[0].Content)
		}
	}

	// At this point, all the tests are done. This means we can clean up after ourselves and delete
	// the collection we created at the beginning. Since all the text objects we made were children of
	// the collection, deleting the collection will recursively delete all the text objects.
	if resp, err := client.DeleteObject(ctx, collection.ID, nil); err != nil {
		t.Fatal(err)
	} else if !resp.Deleted {
		t.Fatal("expected collection to be deleted")
	}

	// Done!
}
