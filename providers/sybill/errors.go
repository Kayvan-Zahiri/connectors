package sybill

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors carry a `detail` that is a message for auth, scope and rate-limit failures,
// and a list of validation problems for 422 responses.
// https://api.sybill.ai/docs/authentication.html#error-responses
// https://api.sybill.ai/docs/rate-limiting.html
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	interpreter.FormatTemplate{
		MustKeys: []string{"detail"},
		Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
	},
)

type ResponseError struct {
	Detail any `json:"detail"`
}

func (r ResponseError) CombineErr(base error) error {
	switch detail := r.Detail.(type) {
	case string:
		if detail == "" {
			return base
		}

		return fmt.Errorf("%w: %s", base, detail)
	case []any:
		messages := make([]string, 0, len(detail))

		for _, entry := range detail {
			if problem, ok := entry.(map[string]any); ok {
				messages = append(messages, describeValidationError(problem))
			}
		}

		if len(messages) == 0 {
			return base
		}

		return fmt.Errorf("%w: %s", base, strings.Join(messages, "; "))
	default:
		return base
	}
}

// describeValidationError renders one HTTPValidationError entry, {"loc": ["query", "limit"], "msg": "..."}.
func describeValidationError(problem map[string]any) string {
	message, _ := problem["msg"].(string)
	parts, _ := problem["loc"].([]any)

	location := make([]string, 0, len(parts))
	for _, part := range parts {
		location = append(location, fmt.Sprint(part))
	}

	if len(location) == 0 {
		return message
	}

	return fmt.Sprintf("%s (%s)", message, strings.Join(location, "."))
}
