package schema

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qraqula/qla/internal/graphql"
)

func strPtr(s string) *string { return &s }

// cannedIntrospectionResponse returns a minimal but realistic introspection
// JSON response containing user-defined types and built-in __ types.
func cannedIntrospectionResponse() json.RawMessage {
	return json.RawMessage(`{
		"data": {
			"__schema": {
				"queryType": {"name": "Query"},
				"mutationType": {"name": "Mutation"},
				"subscriptionType": null,
				"types": [
					{
						"kind": "OBJECT",
						"name": "Query",
						"description": "Root query type",
						"fields": [
							{
								"name": "user",
								"description": "Fetch a user by ID",
								"args": [
									{
										"name": "id",
										"description": "",
										"type": {"kind": "NON_NULL", "name": null, "ofType": {"kind": "SCALAR", "name": "ID", "ofType": null}},
										"defaultValue": null
									}
								],
								"type": {"kind": "OBJECT", "name": "User", "ofType": null},
								"isDeprecated": false,
								"deprecationReason": null
							}
						],
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": []
					},
					{
						"kind": "OBJECT",
						"name": "Mutation",
						"description": "",
						"fields": [
							{
								"name": "createUser",
								"description": "",
								"args": [],
								"type": {"kind": "OBJECT", "name": "User", "ofType": null},
								"isDeprecated": false,
								"deprecationReason": null
							}
						],
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": []
					},
					{
						"kind": "OBJECT",
						"name": "User",
						"description": "A user",
						"fields": [
							{
								"name": "id",
								"description": "",
								"args": [],
								"type": {"kind": "NON_NULL", "name": null, "ofType": {"kind": "SCALAR", "name": "ID", "ofType": null}},
								"isDeprecated": false,
								"deprecationReason": null
							},
							{
								"name": "name",
								"description": "",
								"args": [],
								"type": {"kind": "SCALAR", "name": "String", "ofType": null},
								"isDeprecated": false,
								"deprecationReason": null
							}
						],
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": []
					},
					{
						"kind": "SCALAR",
						"name": "String",
						"description": "Built-in string",
						"fields": null,
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": null
					},
					{
						"kind": "SCALAR",
						"name": "ID",
						"description": "Built-in ID",
						"fields": null,
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": null
					},
					{
						"kind": "OBJECT",
						"name": "__Schema",
						"description": "Introspection schema type",
						"fields": [],
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": []
					},
					{
						"kind": "OBJECT",
						"name": "__Type",
						"description": "Introspection type type",
						"fields": [],
						"inputFields": null,
						"enumValues": null,
						"possibleTypes": null,
						"interfaces": []
					}
				]
			}
		}
	}`)
}

func TestFetchSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is a POST with correct content type.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		// Verify it sends an introspection query.
		var req graphql.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(req.Query) < 50 {
			t.Errorf("expected introspection query, got short query: %q", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(cannedIntrospectionResponse())
	}))
	defer srv.Close()

	client := graphql.NewClient()
	schema, err := FetchSchema(context.Background(), client, srv.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify root types are set.
	if schema.QueryType == nil || schema.QueryType.Name == nil || *schema.QueryType.Name != "Query" {
		t.Errorf("expected QueryType.Name = Query, got %v", schema.QueryType)
	}
	if schema.MutationType == nil || schema.MutationType.Name == nil || *schema.MutationType.Name != "Mutation" {
		t.Errorf("expected MutationType.Name = Mutation, got %v", schema.MutationType)
	}
	if schema.SubscriptionType != nil {
		t.Errorf("expected SubscriptionType = nil, got %v", schema.SubscriptionType)
	}

	// Verify __ types are filtered out.
	for _, typ := range schema.Types {
		if len(typ.Name) >= 2 && typ.Name[:2] == "__" {
			t.Errorf("expected built-in type %q to be filtered out", typ.Name)
		}
	}

	// Verify user-defined types are present.
	expectedTypes := []string{"Query", "Mutation", "User", "String", "ID"}
	for _, name := range expectedTypes {
		if schema.TypeByName(name) == nil {
			t.Errorf("expected type %q to be present", name)
		}
	}

	// Should have exactly 5 types (no __ types).
	if len(schema.Types) != 5 {
		t.Errorf("expected 5 types, got %d", len(schema.Types))
	}

	// Verify fields on the Query type.
	queryType := schema.TypeByName("Query")
	if queryType == nil {
		t.Fatal("Query type not found")
	}
	if len(queryType.Fields) != 1 {
		t.Fatalf("expected 1 field on Query, got %d", len(queryType.Fields))
	}
	if queryType.Fields[0].Name != "user" {
		t.Errorf("expected field name 'user', got %q", queryType.Fields[0].Name)
	}

	// Verify args on Query.user field.
	if len(queryType.Fields[0].Args) != 1 {
		t.Fatalf("expected 1 arg on Query.user, got %d", len(queryType.Fields[0].Args))
	}
	if queryType.Fields[0].Args[0].Name != "id" {
		t.Errorf("expected arg name 'id', got %q", queryType.Fields[0].Args[0].Name)
	}

	// Verify User type has expected fields.
	userType := schema.TypeByName("User")
	if userType == nil {
		t.Fatal("User type not found")
	}
	if len(userType.Fields) != 2 {
		t.Errorf("expected 2 fields on User, got %d", len(userType.Fields))
	}
}

func TestFetchSchemaServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"message":"internal server error"}]}`))
	}))
	defer srv.Close()

	client := graphql.NewClient()
	_, err := FetchSchema(context.Background(), client, srv.URL, nil)
	if err == nil {
		t.Fatal("expected error from server returning 500")
	}
}

func TestFetchSchemaGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":null,"errors":[{"message":"not authorized"}]}`))
	}))
	defer srv.Close()

	client := graphql.NewClient()
	_, err := FetchSchema(context.Background(), client, srv.URL, nil)
	if err == nil {
		t.Fatal("expected error from GraphQL errors in response")
	}
}

func TestIntrospectionQueryContainsTypeRef(t *testing.T) {
	// Verify the introspection query contains the TypeRef fragment
	// for deeply nested type references.
	if len(IntrospectionQuery) < 100 {
		t.Error("introspection query seems too short")
	}
	// Should contain the fragment for nested type wrapping.
	contains := func(s, substr string) bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}
	if !contains(IntrospectionQuery, "TypeRef") {
		t.Error("expected introspection query to contain TypeRef fragment")
	}
	if !contains(IntrospectionQuery, "__schema") {
		t.Error("expected introspection query to contain __schema")
	}
}
