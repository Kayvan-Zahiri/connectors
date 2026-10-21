package clarify

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors follow JSON:API: a top-level `errors` array whose entries carry `status` and `detail`,
// plus `title` and `source.pointer` on validation errors.
// https://developer.clarify.ai/docs/api-basics/errors#error-response-format
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	interpreter.FormatTemplate{
		MustKeys: []string{"errors"},
		Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
	},
)

type ResponseError struct {
	Errors []struct {
		Status string  `json:"status"`
		Title  string  `json:"title,omitempty"`
		Detail *string `json:"detail"`
	} `json:"errors"`
}

func (r ResponseError) CombineErr(base error) error {
	messages := make([]string, 0, len(r.Errors))

	for _, entry := range r.Errors {
		parts := make([]string, 0, 2) // nolint:mnd

		if entry.Title != "" {
			parts = append(parts, entry.Title)
		}

		if entry.Detail != nil && *entry.Detail != "" {
			parts = append(parts, *entry.Detail)
		}

		if len(parts) != 0 {
			messages = append(messages, strings.Join(parts, ": "))
		}
	}

	if len(messages) == 0 {
		return base
	}

	return fmt.Errorf("%w: %s", base, strings.Join(messages, "; "))
}
