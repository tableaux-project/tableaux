package path

import (
	"github.com/tableaux-project/tableaux/config"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/util"
)

type SizeResolver struct {
}

func (sizeResolver SizeResolver) ResolvePathName(columnSchema config.TableSchemaColumn) string {
	path := columnSchema.Path

	// Just append the path - it will be correctly filled via joining
	return util.DescriptorToIdentifier(path + "." + "count_result")
}
