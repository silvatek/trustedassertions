package trust

import (
	"context"
	"errors"
	"testing"
)

func TestGetModelUsesSimple(t *testing.T) {
	resolver := newFakeResolver()
	for _, id := range []string{"", "simple", "unknown"} {
		model, err := GetModel(context.Background(), id, Roots{}, resolver)
		if err != nil {
			t.Fatalf("GetModel(%q): %v", id, err)
		}
		if model.ID() != "simple" {
			t.Errorf("GetModel(%q).ID() = %q, want simple", id, model.ID())
		}
	}
}

func TestGetModelIndependentInstances(t *testing.T) {
	resolver := newFakeResolver()
	a, err := GetModel(context.Background(), "simple", Roots{}, resolver)
	if err != nil {
		t.Fatalf("GetModel a: %v", err)
	}
	b, err := GetModel(context.Background(), "simple", Roots{}, resolver)
	if err != nil {
		t.Fatalf("GetModel b: %v", err)
	}
	if a == b {
		t.Fatal("GetModel returned the same instance twice")
	}
}

func TestGetModelNilResolver(t *testing.T) {
	_, err := GetModel(context.Background(), "simple", Roots{}, nil)
	if !errors.Is(err, ErrNilResolver) {
		t.Errorf("GetModel(nil resolver) error = %v, want ErrNilResolver", err)
	}
}

func TestGetModelIsReady(t *testing.T) {
	d := newDebate(t)
	model, err := GetModel(context.Background(), "simple", Roots{}, d.resolver)
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	_, err = model.Evaluate(context.Background(), d.MoonIsRock)
	if errors.Is(err, ErrNotSetup) {
		t.Fatal("GetModel returned a model that is not set up")
	}
}
