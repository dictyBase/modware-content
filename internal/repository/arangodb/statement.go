package arangodb

const (
	ContentFindBySlug = `
		FOR cnt IN @@content_collection
			FILTER cnt.slug == @slug
			LIMIT 1
			RETURN cnt
	`
	ContentList = `
		FOR cnt IN @@content_collection
			SORT cnt.created_on DESC
			LIMIT @limit
			RETURN cnt
	`
	ContentListWithCursor = `
		FOR cnt IN @@content_collection
			FILTER cnt.created_on <= DATE_ISO8601(@cursor)
			SORT cnt.created_on DESC
			LIMIT @limit
			RETURN cnt
	`
	ContentListFilter = `
		FOR cnt IN @@content_collection
			%s
			SORT cnt.created_on DESC
			LIMIT @limit
			RETURN cnt
	`
	ContentListFilterWithCursor = `
		FOR cnt IN @@content_collection
			FILTER cnt.created_on <= DATE_ISO8601(@cursor)
			%s
			SORT cnt.created_on DESC
			LIMIT @limit
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
