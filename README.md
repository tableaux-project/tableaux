# Tableaux - Structured data retrieval module with pluggable data sources

**Note:** This project is WIP. This is not production ready, and for the most part untested. Please help,
and further this project by contributing!

## About

Tableaux is an abstraction library, which is intended to be used as a simple data extraction backend. It is neither
an ORM, nor a GraphQL clone. Instead, it kind of fills the middle ground, by providing data in a table like matter,
while also giving a lot of freedom on how to retrieve data, e.g. by selecting different columns, applying filtering
and sorting, and even applying extendable schemas.

### State of things

Tableaux is split into 2 projects:

* Tableaux - the actual project with its core mechanics
* Tableaux-Server - a reference implementation for a backend server

Tableaux is meant to have pluggable data sources. That is, have an abstract interface, with different data sources in
the background implementing the interface. Right now, the only data source implemented is the SQL data source. As such,
it is the reference implementation of a Tableaux data source. To make it work, you need to complement it with a database
connection, which right now has reference implementations via Tableaux-MySQL and Tableaux-PSQL

## Getting started

Using Tableaux-Server, it is very easy to get started. Just copy the following code, adjust your database and credentials,
and there you go - you got yourself a Tableaux backend server up and running!

```go
package main

import (
    "time"

    "gopkg.in/birkirb/loggers.v1/log"

    "github.com/tableaux-project/tableaux"
    "github.com/tableaux-project/tableaux/datasource/sqlsource"
    "github.com/tableaux-project/tableauxmysql"
    "github.com/tableaux-project/tableauxserver"
)

func main() {
    start := time.Now()

    enumMapper, err := tableaux.NewEnumMapperFromAssets()
    if err != nil {
        log.Fatal(err)
    }

    translator, err := tableaux.NewTranslatorFromAssets()
    if err != nil {
        log.Fatal(err)
    }

    schemaMapper, err := tableaux.NewSchemaMapperFromAssets()
    if err != nil {
        log.Fatal(err)
    }

    // ----------------------

    var (
        databaseAddress  = "localhost"
        databasePort     = 3306
        databaseUser     = "someUser"
        databasePassword = "123"
        databaseName     = "myDatabase"
    )

    // ----------------------

    mysqlconnector := tableauxmysql.NewMySQLDatabaseConnector(databaseAddress, databasePort, databaseName, databaseUser, databasePassword)
    defer mysqlconnector.Close()

    //psqlconnector := tableauxpsql.NewPSQLDatabaseConnector("tc-postgresql", 5432, databaseName, databaseUser, databasePassword)
    //defer psqlconnector.Close()

    // ----------------------

    dataSourceConnector, err := sqlsource.NewConnector(mysqlconnector, enumMapper, translator, schemaMapper)
    if err != nil {
        log.Fatal(err)
    }

    serverHandler, err := tableauxserver.NewServer(dataSourceConnector, schemaMapper, 8081)
    if err != nil {
        log.Fatal(err)
    }
    log.Fatal(serverHandler.ListenAndServe())
}
```

## JSON configuration

For the library to work, three components must be loaded. A Translator, an EnumMapper, and a SchemaMapper. While it is possible to use
in-code versions, Tableaux provides means to load these by json configuration files.

You can either use the `New...FromFolder` methods in the config package of Tableaux, or you can use the recommended approach, which is to
use the `New..FromAssetsFolder` methods in the Tableaux base package (see example above). The latter assumes configuration files relative
to the executing binary.

Now, for more details on the actual configuration and their files:

### Schema files

The schema folder provides the so called schema files. Schema files provide the blueprint for any table
configuration accessible.

Schema files are exposed to the consumer via http endpoints, so configuration for data requests can be built dynamically on the consumer side.

The schema folder is traversed in a recursive fashion, which allows to structure schema files in any fashion
desired. Note however, that the folder structure is reflected in both extensions, and the path under which the
schema will be made accessible via http endpoint.

```json
{
  "entity": "companyDivision",
  "extensions": [
    {
      "title": "columns.abstract.system",
      "table": "abstract_entity",
      "key": null
    },
    {
      "title": "columns.masterdata.company",
      "table": "companies",
      "key": "company"
    }
  ],
  "exclusions": [
    "companyDivision_company_companyDivisions",
  ],
  "columns": [
    {
      "title": "columns.masterdata.companydivision.divisionid",
      "path": "companyDivision_divisionId",
      "type": "long",
      "filter": "NumericFilter",
      "order": "NumericOrder",
      "frontendHints": {
        "showDefault": false
      }
    },
    {
      "title": "columns.masterdata.companydivision.name",
      "path": "companyDivision_name",
      "type": "string",
      "filter": "StringRegExFilter",
      "order": "StringOrder",
      "frontendHints": {
        "showDefault": true
      }
    }
  ]
}
```

The `entity` key tells the name of the corresponding table in the database. `extensions` defines extensions (or relations) which are to be included,
`exclusions` lists path prefixes which should be removed after extensions are applied. `columns` finally contains the definition of the individual columns.

#### Columns

`columns` is an array of individual columns. An example:

```json
    {
      "title": "columns.masterdata.companydivision.divisionid",
      "path": "companyDivision_divisionId",
      "type": "long",
      "filter": "NumericFilter",
      "order": "NumericOrder",
      "frontendHints": {
        "showDefault": false
      }
    }
```

Key | Description
--- | ---
title | The title of the column. Could be plain text or a translation key. Tableaux will **not** try translate the key though, as that is the responsibility of the consumer.
path | The underscore delimited path, which leads to individual attributes. More on this in paragraph **Path resolving**.
type | The type of the column, which can either be a primitive, or the name of an enum.
filter | Currently not used ; used to be the filter component which translated filter input strings to type safe filtering the database could understand.
order | Order component which tells the library how a column should be ordered. This can be used to handle special cases such as enum ordering.
frontendHints | Optional frontend hints, e.g. if the column should be shown per default. Tableaux does not use or process these hints in any way.

#### Extensions

Extensions provide a powerful way of integrating schema files into each other. This helps keep the schema definition [DRY](http://wiki.c2.com/?DontRepeatYourself).

Extensions can be used to describe a relation between two tables, which allows to dynamically join information of related entity together at runtime. For example,
a `user` schema could provide an extension to a `usergroup` schema, which allows the user table to include both information about users, and their respective user
groups at runtime (assuming the relation is n:1 in this example). Since the resolving of entity relations is handled dynamically from the database schema, it is
required that these entities have proper relations set up in the database for this to work (otherwise - this will fail at **runtime when requesting data**).

```json
      {
        "title": "columns.abstract.system",
        "table": "abstract_entity",
        "key": null
      }
```

Key | Description
--- | ---
title | The title of the extension. Could be plain text or a translation key. The library will **not** try translate the key though, as that is the responsibility of the consumer. This key can be used to group columns into logical groups on the consumer side.
table | The path to the schema file which should be the base for the extension (note that the .json suffix **must** be omitted)
key | The substitution key, which is appended to the `entity` key of the schema for path substitution (see explanation below).

More information about the substitution mechanism:

Substitution works by calculating a substitution key, which is then used to replace the first part of all columns that are referenced in the extensions schema columns.

For example, take the path `person_personKey` in a schema file `persons` which is to be used as an extension.
If in the schema to be used `entity` is `organization`, and the `key` of the extension is `assignedPerson`, this will yield a substitution key of `organization_assignedPerson`,
and thus the replaced path will be `organization_assignedPerson_personKey`. This mechanism works recursively, and works even when `persons` has itself extensions.

`key` can be *null*, which makes the substitution key solly the name of the `entity`. This can be used, to *embed* schemas into the schema, without constructing a nested path.

For example, take the path `abstractEntity_createDateUtc` in a schema file `abstract_entity` which is to be used as an extension.
If in the schema to be used `entity` is `organization`, and the `key` of the extension is `null`, this will yield a substitution key of `organization`,
and thus the replaced path will be `organization_createDateUtc`. This mechanism is great to extract common attributes which are shared by multiple schema files.

#### Exclusions

The `exclusions` key is an array of paths, which are to be excluded after extensions are applied. Due to how extensions work, very long and sometimes even duplicated paths can
occur. For example, `organization_supervisor` and `organization_assignedPerson_supervisor` might reference the same supervisor. So it makes sense to exclude one of them, so the
columns are not cluttered with redundant information. All paths, and their sub-paths, are excluded by the provided keys. So, `organization_assignedPerson_supervisor` as an
excluded path will exclude all paths that are equal and/or below it, e.g. `organization_assignedPerson_supervisor_supervisorKey`. Note, that this mechanism does NOT prevent
recursion, which might crash the program!

### Enums

TODO

### i18n

TODO

## Logging

This library uses [loggers](https://github.com/birkirb/loggers/) for logging abstraction. This means, that is is easy to plug-in your prefered logging library of choice.

If you want to use [Logrus](https://github.com/sirupsen/logrus/) for example, the following code snippet should get you started:

```go
import (
    mapper "github.com/birkirb/loggers-mapper-logrus"
    "github.com/sirupsen/logrus"
    "gopkg.in/birkirb/loggers.v1/log"
)

func init() {
    l := logrus.New()
    l.Out = os.Stdout

    l.Formatter = &logrus.TextFormatter{
        ForceColors:      true,
        DisableTimestamp: false,
        FullTimestamp:    true,
        TimestampFormat:  "2006/01/02 15:04:05",
    }
    l.Level = logrus.DebugLevel

    log.Logger = mapper.NewLogger(l)
}
```

## Dependencies and licensing

Tableaux is licenced via MIT, as specified [here](https://github.com/tableaux-project/tableaux/blob/master/LICENSE).

* [loggers](https://github.com/birkirb/loggers/) - [MIT](https://github.com/birkirb/loggers/blob/master/LICENSE.txt) - Abstract logging for Golang projects

## Versioning

We use [SemVer](http://semver.org/) for versioning.
