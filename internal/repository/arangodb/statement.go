package arangodb

const (
	ContentFindBySlug = `
		FOR cnt IN @@content_collection
			FILTER cnt.slug == @slug
			LIMIT 1
			RETURN cnt
	`

	ContentInsert = `
		INSERT {
			name: @name,
			slug: @slug,
			namespace: @namespace,
			created_by: @created_by,
			updated_by: @updated_by,
			content: @content,
			created_on : DATE_ISO8601(DATE_NOW()),
			updated_on : DATE_ISO8601(DATE_NOW()),
		} INTO @@content_collection RETURN NEW
	`

	ContentUpdate = `
		UPDATE { 
			_key: @key, 
			updated_by: @updated_by, 
			updated_on : DATE_ISO8601(DATE_NOW()),
			content: @content 
		} IN @@content_collection RETURN NEW
	`
)
