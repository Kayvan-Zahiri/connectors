package circleback

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/circleback/internal/metadata"
)

// ListObjectMetadata serves the catalog generated from the OpenAPI spec, adding the meetings
// transcript that Read fetches from its own endpoint.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	selection, err := metadata.Catalog.Select(objectNames)
	if err != nil {
		return nil, err
	}

	// Segments of {speaker, text, timestamp}: the speaker's name, the words spoken and the
	// segment's start in seconds from the beginning of the meeting.
	selection.AddField(objectNameMeetings, fieldTranscript, common.FieldMetadata{
		DisplayName:  "Transcript",
		ValueType:    common.ValueTypeOther,
		ProviderType: "array",
	})

	return selection.Result(), nil
}
