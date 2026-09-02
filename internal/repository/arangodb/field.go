package arangodb

// Bind variable and field names shared by queries and index definitions.
const (
	keyContentCollection = "@content_collection"
	keyNamespace         = "namespace"
	keySlug              = "slug"
)

// FilterMap provides mapping of filter attributes to database fields.
func FilterMap() map[string]string {
	return map[string]string{
		"name":       "cnt.name",
		keyNamespace: "cnt.namespace",
		keySlug:      "cnt.slug",
		"created_by": "cnt.created_by",
	}
}
