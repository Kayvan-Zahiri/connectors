package circleback

import (
	"fmt"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors are `{"error": "<message>", "code": "<CODE>"}`, with field-level `issues` on validation failures.
// https://circleback.ai/docs/api#rate-limits (components.schemas.ErrorResponse in the OpenAPI spec)
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	interpreter.FormatTemplate{
		MustKeys: []string{"error"},
		Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
	},
)

type ResponseError struct {
	Message string `json:"error"`
	Code    string `json:"code,omitempty"`
	Issues  []struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	} `json:"issues,omitempty"`
}

func (r ResponseError) CombineErr(base error) error {
	message := r.Message
	if r.Code != "" {
		message = fmt.Sprintf("%s (%s)", message, r.Code)
	}

	for _, issue := range r.Issues {
		message = fmt.Sprintf("%s; %s: %s", message, issue.Path, issue.Message)
	}

	if message == "" {
		return base
	}

	return fmt.Errorf("%w: %s", base, message)
}
