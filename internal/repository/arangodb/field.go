package arangodb

// FilterMap provides mapping of filter attributes to database fields.
func FilterMap() map[string]string {
	return map[string]string{
		"name":       "cnt.name",
		"namespace":  "cnt.namespace",
		"slug":       "cnt.slug",
		"created_by": "cnt.created_by",
	}
}
