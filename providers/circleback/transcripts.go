package circleback

import (
	"context"
	"sync"

	"github.com/amp-labs/amp-common/simultaneously"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

const (
	// fieldTranscript is the meetings field filled from GET /meeting/{meetingId}/transcript.
	// The list of meetings doesn't carry the transcript.
	// https://circleback.ai/docs/api/openapi.json (paths./meeting/{meetingId}/transcript.get)
	fieldTranscript = "transcript"

	// The Free plan allows 3 requests per second, across every key of the account.
	// https://circleback.ai/docs/api#rate-limits
	maxConcurrentTranscriptFetches = 2
)

// transcriptRegistry maps a meeting ID to its transcript segments.
type transcriptRegistry map[string]any

// withTranscripts fetches the transcript of every meeting the filter keeps,
// so a meeting outside the Since/Until window costs no extra request.
func (c *Connector) withTranscripts(
	ctx context.Context, filter common.RecordsFilterFunc, transcripts transcriptRegistry,
) common.RecordsFilterFunc {
	return func(params common.ReadParams, body *ajson.Node, records []*ajson.Node) ([]*ajson.Node, string, error) {
		filtered, next, err := filter(params, body, records)
		if err != nil {
			return nil, "", err
		}

		var mutex sync.Mutex

		jobs := make([]simultaneously.Job, 0, len(filtered))

		for _, record := range filtered {
			meetingID, err := jsonquery.New(record).StringRequired("id")
			if err != nil {
				return nil, "", err
			}

			jobs = append(jobs, func(ctx context.Context) error {
				transcript, err := c.fetchTranscript(ctx, meetingID)
				if err != nil {
					return err
				}

				mutex.Lock()
				defer mutex.Unlock()

				transcripts[meetingID] = transcript

				return nil
			})
		}

		if err := simultaneously.DoCtx(ctx, maxConcurrentTranscriptFetches, jobs...); err != nil {
			return nil, "", err
		}

		return filtered, next, nil
	}
}

func (c *Connector) fetchTranscript(ctx context.Context, meetingID string) (any, error) {
	transcriptURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, "meeting", meetingID, fieldTranscript)
	if err != nil {
		return nil, err
	}

	response, err := c.JSONHTTPClient().Get(ctx, transcriptURL.String())
	if err != nil {
		return nil, err
	}

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	segments, err := jsonquery.New(body).ArrayRequired("")
	if err != nil {
		return nil, err
	}

	return jsonquery.Convertor.ArrayToObjects(segments)
}

// embed returns the meeting's fields with its transcript added. Raw stays the listed meeting.
func (t transcriptRegistry) embed(node *ajson.Node) (map[string]any, error) {
	record, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return nil, err
	}

	meetingID, _ := record["id"].(string)
	if transcript, ok := t[meetingID]; ok {
		record[fieldTranscript] = transcript
	}

	return record, nil
}
