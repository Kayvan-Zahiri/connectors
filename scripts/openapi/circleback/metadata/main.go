package main

import (
	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/circleback/internal/files"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/filters"
	mapping "github.com/amp-labs/connectors/scripts/openapi/internal/api/map"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/map/mappingfields"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/output"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/pipeline"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/reducers"
)

func main() {
	extractor, err := files.OpenApiFile.Extractor()
	goutils.MustBeNil(err)

	// Every Circleback list endpoint returns a bare JSON array.
	getSchemas, err := extractor.ExtractListSchemas(api.GET)
	goutils.MustBeNil(err)

	// Only the collections this connector reads.
	readObjects := filters.NewRules(
		"/meetings",
		"/action-items",
		"/tag",
		"/people",
		"/companies",
	)

	pipe := pipeline.NewSchemaPipe(getSchemas).
		Filter(filters.KeepWithPath(filters.AndPathMatcher{
			filters.NoIdPath{}, filters.NewAllowPathStrategy(readObjects),
		})).
		Reduce(reducers.ShortestNameFromUrlWithFullFallback).
		Map(mapping.DisplayNameFromObjectName).
		Map(mapping.DisplayNameFormat(
			mapping.CamelCaseToSpaceSeparated,
			mapping.CapitalizeFirstLetterEveryWord,
		)).
		Map(mapping.Fields(
			nullableField,
			mappingfields.DisplayNameFormat(func(_, name string) string {
				return mapping.CapitalizeFirstLetterEveryWord(mapping.CamelCaseToSpaceSeparated(name))
			}),
		))

	output.PrintUnusedRules("readObjects", readObjects)
	goutils.MustBeNil(output.WriteMetadata(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteQueryParamStats(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteEndpoints(files.OutputDir, &pipe, nil, nil, nil))
}
