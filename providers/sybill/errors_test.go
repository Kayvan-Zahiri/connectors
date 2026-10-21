package sybill

import (
	"errors"
	"testing"

	"github.com/amp-labs/connectors/common"
)

func TestResponseErrorCombineErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		detail any
		want   string
	}{
		{
			name:   "Message detail is appended",
			detail: "Conversation not found",
			want:   "not found: Conversation not found",
		},
		{
			name: "Validation problems are listed with their location",
			detail: []any{
				map[string]any{"loc": []any{"query", "limit"}, "msg": "Input should be less than or equal to 50"},
			},
			want: "not found: Input should be less than or equal to 50 (query.limit)",
		},
		{
			name:   "Missing detail keeps the base error",
			detail: nil,
			want:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ResponseError{Detail: tt.detail}.CombineErr(common.ErrNotFound)
			if !errors.Is(err, common.ErrNotFound) {
				t.Fatalf("expected %v to wrap %v", err, common.ErrNotFound)
			}

			if err.Error() != tt.want {
				t.Fatalf("got %q, want %q", err.Error(), tt.want)
			}
		})
	}
}
