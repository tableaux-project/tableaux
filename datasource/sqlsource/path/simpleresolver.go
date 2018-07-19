package path

import (
	"strings"

	"github.com/tableaux-project/tableaux/config"
	"github.com/tableaux-project/tableaux/datasource/sqlsource/util"
)

type SimpleResolver struct {
}

func (simpleResolver SimpleResolver) ResolvePathName(columnSchema config.TableSchemaColumn) string {
	path := columnSchema.Path

	pathParts := strings.Split(path, "_")
	var joinedPath = pathParts[0]

	if len(pathParts) > 1 {
		joinedPath = util.DescriptorToIdentifier(strings.Join(pathParts[0:len(pathParts)-1], "_") + "." + pathParts[len(pathParts)-1])
	}

	return joinedPath
}
