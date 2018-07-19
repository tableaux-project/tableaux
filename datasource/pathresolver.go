package datasource

import "github.com/tableaux-project/tableaux/config"

type PathResolver interface {
	ResolvePathName(columnSchema config.TableSchemaColumn) string
}
