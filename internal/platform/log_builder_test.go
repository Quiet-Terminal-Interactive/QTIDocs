package platform

import (
	"context"
	"testing"
)

func TestLogBuilder_Enqueue(t *testing.T) {
	var b LogBuilder
	if err := b.Enqueue(context.Background(), BuildJob{Subdomain: "acme"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogBuilder_Teardown(t *testing.T) {
	var b LogBuilder
	if err := b.Teardown(context.Background(), "acme"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
