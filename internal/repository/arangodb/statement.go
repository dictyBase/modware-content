package arangodb

const (
	ContentFindBySlug = `
		FOR cnt IN @@content_collection
			FILTER cnt.slug == @slug
			LIMIT 1
			RETURN cnt
	`
)
