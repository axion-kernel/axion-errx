package errx_test

import (
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

func TestNewTypeCarriesNameAndCode(t *testing.T) {
	t.Parallel()

	typ := errx.NewType("type_test_new", 990101)
	if typ.Name() != "type_test_new" {
		t.Fatalf("unexpected type name: got %q", typ.Name())
	}
	if typ.Code() != 990101 {
		t.Fatalf("unexpected type code: got %d", typ.Code())
	}
}

func TestTypeAccessorsHandleNilReceiver(t *testing.T) {
	t.Parallel()

	var typ *errx.Type
	if typ.Name() != "" {
		t.Fatalf("unexpected nil type name: %q", typ.Name())
	}
	if typ.Code() != 0 {
		t.Fatalf("unexpected nil type code: %d", typ.Code())
	}
}
