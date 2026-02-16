package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qraqula/qla/internal/graphql"
)

// IntrospectionQuery is the standard GraphQL introspection query with
// the TypeRef fragment for deeply nested NON_NULL/LIST wrapping (7 levels).
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
    }
  }
}
`

// introspectionResponse wraps the raw JSON {"__schema": {...}} data
// returned in the GraphQL response.
type introspectionResponse struct {
	Schema Schema `json:"__schema"`
}

// FetchSchema sends the standard GraphQL introspection query to the given
// endpoint using the provided client. It parses the response into a Schema,
// filtering out built-in introspection types (those prefixed with "__").
func FetchSchema(ctx context.Context, client *graphql.Client, endpoint string) (*Schema, error) {
	req := graphql.Request{
		Query: IntrospectionQuery,
	}

	result, err := client.Execute(ctx, endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("introspection request: %w", err)
	}

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return nil, fmt.Errorf("introspection failed with status %d", result.StatusCode)
	}

	if result.Response.HasErrors() {
		msgs := make([]string, len(result.Response.Errors))
		for i, e := range result.Response.Errors {
			msgs[i] = e.Message
		}
		return nil, fmt.Errorf("introspection errors: %s", strings.Join(msgs, "; "))
	}

	var wrapper introspectionResponse
	if err := json.Unmarshal(result.Response.Data, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal introspection response: %w", err)
	}

	// Filter out built-in introspection types (prefixed with "__").
	filtered := make([]FullType, 0, len(wrapper.Schema.Types))
	for _, t := range wrapper.Schema.Types {
		if !strings.HasPrefix(t.Name, "__") {
			filtered = append(filtered, t)
		}
	}
	wrapper.Schema.Types = filtered

	return &wrapper.Schema, nil
}
