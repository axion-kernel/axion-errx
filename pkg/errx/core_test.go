package errx_test

import (
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

func TestCoreErrorCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		typ  *errx.Type
		code int
	}{
		{name: "internal_error", typ: errx.InternalError, code: 1000},
		{name: "illegal_argument", typ: errx.IllegalArgument, code: 1001},
		{name: "unauthorized", typ: errx.Unauthorized, code: 1002},
		{name: "forbidden", typ: errx.Forbidden, code: 1003},
		{name: "not_found", typ: errx.NotFound, code: 1004},
		{name: "conflict", typ: errx.Conflict, code: 1005},
		{name: "data_unavailable", typ: errx.DataUnavailable, code: 1006},
		{name: "rejected_operation", typ: errx.RejectedOperation, code: 1007},
		{name: "unsupported_operation", typ: errx.UnsupportedOperation, code: 1008},
		{name: "illegal_state", typ: errx.IllegalState, code: 1009},
		{name: "illegal_format", typ: errx.IllegalFormat, code: 1010},
		{name: "external_error", typ: errx.ExternalError, code: 1011},
		{name: "initialization_failed", typ: errx.InitializationFailed, code: 1012},
		{name: "method_not_allowed", typ: errx.MethodNotAllowed, code: 1013},
		{name: "bad_request_body", typ: errx.BadRequestBody, code: 1014},
		{name: "upstream_disconnected", typ: errx.UpstreamDisconnected, code: 1015},
		{name: "upstream_quota_exhausted", typ: errx.UpstreamQuotaExhausted, code: 1016},
		{name: "execution_skipped", typ: errx.ExecutionSkipped, code: 1017},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.typ.Name() != tc.name {
				t.Fatalf("unexpected type name: got %q want %q", tc.typ.Name(), tc.name)
			}
			if tc.typ.Code() != tc.code {
				t.Fatalf("unexpected code: got %d want %d", tc.typ.Code(), tc.code)
			}
		})
	}
}

func TestCoreErrorsCrossMatchWithIsOfType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		typ  *errx.Type
	}{
		{name: "internal_error", typ: errx.InternalError},
		{name: "illegal_argument", typ: errx.IllegalArgument},
		{name: "unauthorized", typ: errx.Unauthorized},
		{name: "forbidden", typ: errx.Forbidden},
		{name: "not_found", typ: errx.NotFound},
		{name: "conflict", typ: errx.Conflict},
		{name: "data_unavailable", typ: errx.DataUnavailable},
		{name: "rejected_operation", typ: errx.RejectedOperation},
		{name: "unsupported_operation", typ: errx.UnsupportedOperation},
		{name: "illegal_state", typ: errx.IllegalState},
		{name: "illegal_format", typ: errx.IllegalFormat},
		{name: "external_error", typ: errx.ExternalError},
		{name: "initialization_failed", typ: errx.InitializationFailed},
		{name: "method_not_allowed", typ: errx.MethodNotAllowed},
		{name: "bad_request_body", typ: errx.BadRequestBody},
		{name: "upstream_disconnected", typ: errx.UpstreamDisconnected},
		{name: "upstream_quota_exhausted", typ: errx.UpstreamQuotaExhausted},
		{name: "execution_skipped", typ: errx.ExecutionSkipped},
	}

	for _, source := range cases {
		source := source
		t.Run(source.name, func(t *testing.T) {
			t.Parallel()

			err := source.typ.New("test message")
			for _, target := range cases {
				target := target

				got := errx.IsOfType(err, target.typ)
				want := source.typ == target.typ
				if got != want {
					t.Fatalf("unexpected IsOfType result: source=%s target=%s got=%t want=%t", source.name, target.name, got, want)
				}
			}
		})
	}
}

func TestNotImplementedAliasesUnsupportedOperation(t *testing.T) {
	t.Parallel()

	if errx.NotImplemented != errx.UnsupportedOperation {
		t.Fatal("expected not implemented to alias unsupported operation")
	}
}

func TestNotImplementedMatchesUnsupportedOperationWithIsOfType(t *testing.T) {
	t.Parallel()

	err := errx.NotImplemented.New("feature pending")
	if !errx.IsOfType(err, errx.UnsupportedOperation) {
		t.Fatal("expected not implemented error to match unsupported operation")
	}
	if !errx.IsOfType(err, errx.NotImplemented) {
		t.Fatal("expected not implemented error to match not implemented alias")
	}
}
