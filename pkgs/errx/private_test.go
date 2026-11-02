package errx_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/axion-kernel/axion-errx/pkgs/errx"
)

func TestWrapPrivatePreservesCauseWithoutRendering(t *testing.T) {
	cause := &url.Error{Op: "Get", URL: "https://example.test/?token=secret", Err: context.DeadlineExceeded}
	failure := errx.ExternalError.WrapPrivate(cause, "request %d failed", 7)
	var original *url.Error
	if !errors.Is(failure, context.DeadlineExceeded) || !errors.As(failure, &original) || original != cause || failure.Cause() != cause {
		t.Fatal("private cause identity lost")
	}
	if !errx.IsOfType(failure, errx.ExternalError) || failure.Message() != "request 7 failed" || failure.Stacktrace() == "" {
		t.Fatalf("structured identity lost: %v", failure)
	}
	for _, display := range []string{failure.Error(), failure.Message(), fmt.Sprint(failure.Attrs()), fmt.Sprint(errx.Attrs(failure)), errx.WrapError(failure, "outer").Error()} {
		if strings.Contains(display, "secret") || strings.Contains(display, "example.test") {
			t.Fatalf("private cause rendered: %s", display)
		}
	}
	if errx.ExternalError.WrapPrivate(nil, "unused") != nil {
		t.Fatal("nil cause must remain nil")
	}
}
