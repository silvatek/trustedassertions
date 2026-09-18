package trust

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
	refs "silvatek.uk/trustedassertions/internal/references"
)

type fakeResolver struct {
	assertions.NullResolver
	refsByTarget map[string][]refs.Reference
	assertions   map[string]assertions.Assertion
	refsErr      error
}

func newFakeResolver() *fakeResolver {
	return &fakeResolver{
		refsByTarget: make(map[string][]refs.Reference),
		assertions:   make(map[string]assertions.Assertion),
	}
}

func (r *fakeResolver) addAssertion(target, source refs.HashUri, a assertions.Assertion) {
	key := target.Unadorned()
	r.refsByTarget[key] = append(r.refsByTarget[key], refs.Reference{Source: source, Target: target})
	r.assertions[source.Unadorned()] = a
}

func (r *fakeResolver) addRef(target, source refs.HashUri) {
	key := target.Unadorned()
	r.refsByTarget[key] = append(r.refsByTarget[key], refs.Reference{Source: source, Target: target})
}

func (r *fakeResolver) FetchRefs(ctx context.Context, key refs.HashUri) ([]refs.Reference, error) {
	if r.refsErr != nil {
		return nil, r.refsErr
	}
	if incoming, ok := r.refsByTarget[key.Unadorned()]; ok {
		return incoming, nil
	}
	return []refs.Reference{}, nil
}

func (r *fakeResolver) FetchAssertion(ctx context.Context, key refs.HashUri) (assertions.Assertion, error) {
	if a, ok := r.assertions[key.Unadorned()]; ok {
		return a, nil
	}
	return assertions.Assertion{}, errors.New("assertion not found")
}

func makeAssertion(issuer string, category assertions.AssertionType, confidence float32) assertions.Assertion {
	a := assertions.NewAssertion(category)
	a.Issuer = issuer
	a.Confidence = confidence
	return a
}

func uri(hash, kind string) refs.HashUri {
	return refs.MakeUri(hash, kind)
}

func setupSimple(t *testing.T, resolver assertions.Resolver, roots Roots) *SimpleTrustModel {
	t.Helper()
	model := NewSimpleTrustModel()
	if err := model.Setup(context.Background(), resolver, roots); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return model
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// debate is a tiny cast for trust-model tests.
// Neutral always has assertions about both statements; the model must ignore them.
type debate struct {
	t                                 *testing.T
	Honest, Neutral, Scammer, Trusted refs.HashUri
	MoonIsRock, MoonIsCheese          refs.HashUri
	resolver                          *fakeResolver
}

func newDebate(t *testing.T) *debate {
	t.Helper()
	d := &debate{
		t:            t,
		Honest:       uri("honest", "entity"),
		Neutral:      uri("neutral", "entity"),
		Scammer:      uri("scammer", "entity"),
		Trusted:      uri("trusted", "entity"),
		MoonIsRock:   uri("moon-rock", "statement"),   // The moon is made of rock
		MoonIsCheese: uri("moon-cheese", "statement"), // The moon is made of cheese
		resolver:     newFakeResolver(),
	}
	d.affirm(d.Neutral, d.MoonIsRock, 0.15)
	d.refute(d.Neutral, d.MoonIsCheese, 0.25)
	return d
}

func (d *debate) setup(parts ...Roots) *SimpleTrustModel {
	d.t.Helper()
	roots := make(Roots)
	for _, part := range parts {
		for k, v := range part {
			roots[k] = v
		}
	}
	seen := make(map[float64]string)
	for k, v := range roots {
		if refs.UriFromString(k).Unadorned() == d.Neutral.Unadorned() {
			d.t.Fatal("Neutral must not be a trust root")
		}
		if other, ok := seen[v]; ok {
			d.t.Fatalf("duplicate root value %v for %s and %s", v, other, k)
		}
		seen[v] = k
	}
	return setupSimple(d.t, d.resolver, roots)
}

func (d *debate) assertTrustScore(model TrustModel, statement refs.HashUri, want float64) {
	d.t.Helper()
	score, err := model.Evaluate(context.Background(), statement)
	if err != nil {
		d.t.Fatalf("Evaluate: %v", err)
	}
	if !almostEqual(score, want) {
		d.t.Errorf("score = %v, want %v", score, want)
	}
}

func (d *debate) assertUndefined(model TrustModel, statement refs.HashUri) {
	d.t.Helper()
	score, err := model.Evaluate(context.Background(), statement)
	if !errors.Is(err, ErrUndefined) {
		d.t.Errorf("Evaluate error = %v, want ErrUndefined (score %v)", err, score)
	}
}

func (d *debate) affirm(who, about refs.HashUri, confidence float32) {
	d.claim(who.Unadorned(), about, assertions.IsTrue, confidence)
}

func (d *debate) refute(who, about refs.HashUri, confidence float32) {
	d.claim(who.Unadorned(), about, assertions.IsFalse, confidence)
}

func (d *debate) claim(issuer string, about refs.HashUri, category assertions.AssertionType, confidence float32) {
	id := fmt.Sprintf("assertion-%d", len(d.resolver.assertions))
	d.resolver.addAssertion(about, uri(id, "assertion"), makeAssertion(issuer, category, confidence))
}

func trust(entity refs.HashUri, level float64) Roots {
	return Roots{entity.Unadorned(): level}
}

func TestTrustScoringScenarios(t *testing.T) {
	t.Run("very simple debate", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		model := d.setup(trust(d.Honest, 0.5))
		// Honest w=0.5×0.8=0.4; true camp only → P=T=0.4
		d.assertTrustScore(model, d.MoonIsRock, 0.4)
	})

	t.Run("only Honest is trusted", func(t *testing.T) {
		d := newDebate(t)
		model := d.setup(trust(d.Honest, 0.9))

		t.Run("true camp only", func(t *testing.T) {
			d.affirm(d.Honest, d.MoonIsRock, 0.8)
			d.refute(d.Scammer, d.MoonIsRock, 0.6)
			// Honest w=0.9×0.8=0.72; Scammer ignored; true camp only → P=T=0.72
			d.assertTrustScore(model, d.MoonIsRock, 0.72)
		})
		t.Run("false camp only", func(t *testing.T) {
			d.refute(d.Honest, d.MoonIsCheese, 1.0)
			d.affirm(d.Scammer, d.MoonIsCheese, 0.5)
			// Honest F=0.9×1.0=0.9; Scammer ignored; false camp only → P=1-F=0.1
			d.assertTrustScore(model, d.MoonIsCheese, 0.1)
		})
	})

	t.Run("Honest and Scammer cancel", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		d.refute(d.Scammer, d.MoonIsRock, 0.9)
		model := d.setup(trust(d.Honest, 0.9), trust(d.Scammer, 0.8))
		// Honest T=0.9×0.8=0.72; Scammer F=0.8×0.9=0.72; T=F → P=0.5
		d.assertTrustScore(model, d.MoonIsRock, 0.5)
	})

	t.Run("Honest outweighs Scammer", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		d.refute(d.Scammer, d.MoonIsRock, 0.8)
		model := d.setup(trust(d.Honest, 0.9), trust(d.Scammer, 0.4))
		// Honest T=0.9×0.8=0.72; Scammer F=0.4×0.8=0.32;
		// P=T(1-F)/[T(1-F)+F(1-T)]=0.4896/0.5792=0.845304
		d.assertTrustScore(model, d.MoonIsRock, 0.845304)
	})

	t.Run("Honest and Trusted both support rock", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		d.affirm(d.Trusted, d.MoonIsRock, 0.6)
		model := d.setup(trust(d.Honest, 0.9), trust(d.Trusted, 0.7))
		// Honest w=0.9×0.8=0.72; Trusted w=0.7×0.6=0.42;
		// T=1-(1-0.72)(1-0.42)=0.8376; true camp only → P=T=0.8376
		d.assertTrustScore(model, d.MoonIsRock, 0.8376)
	})

	t.Run("Honest and Trusted both refute cheese", func(t *testing.T) {
		d := newDebate(t)
		d.refute(d.Honest, d.MoonIsCheese, 0.8)
		d.refute(d.Trusted, d.MoonIsCheese, 0.5)
		model := d.setup(trust(d.Honest, 0.9), trust(d.Trusted, 0.8))
		// Honest w=0.9×0.8=0.72; Trusted w=0.8×0.5=0.4;
		// F=1-(1-0.72)(1-0.4)=0.832; false camp only → P=1-F=0.168
		d.assertTrustScore(model, d.MoonIsCheese, 0.168)
	})

	t.Run("Honest and Trusted vs Scammer", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		d.affirm(d.Trusted, d.MoonIsRock, 0.6)
		d.refute(d.Scammer, d.MoonIsRock, 0.5)
		model := d.setup(trust(d.Honest, 0.9), trust(d.Trusted, 0.7), trust(d.Scammer, 0.8))
		// Honest w=0.72, Trusted w=0.42 → T=1-(1-0.72)(1-0.42)=0.8376;
		// Scammer F=0.8×0.5=0.4;
		// P=T(1-F)/[T(1-F)+F(1-T)]=0.50256/0.56752=0.885537
		d.assertTrustScore(model, d.MoonIsRock, 0.885537)
	})
}

func TestSameDirectionClaimsCombine(t *testing.T) {
	d := newDebate(t)
	d.affirm(d.Honest, d.MoonIsRock, 0.7)
	d.affirm(d.Trusted, d.MoonIsRock, 0.5)
	d.refute(d.Scammer, d.MoonIsRock, 0.4)
	model := d.setup(trust(d.Honest, 0.9), trust(d.Trusted, 0.8))
	// Honest w=0.9×0.7=0.63; Trusted w=0.8×0.5=0.4; Scammer ignored;
	// T=1-(1-0.63)(1-0.4)=0.778; true camp only → P=T=0.778
	d.assertTrustScore(model, d.MoonIsRock, 0.778)
}

func TestProbabilityTrue(t *testing.T) {
	t.Run("neither camp is undefined", func(t *testing.T) {
		_, err := probabilityTrue(0, false, 0, false)
		if !errors.Is(err, ErrUndefined) {
			t.Errorf("error = %v, want ErrUndefined", err)
		}
	})

	t.Run("true camp only uses T", func(t *testing.T) {
		p, err := probabilityTrue(0.4, true, 0.9, false)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.4) {
			t.Errorf("P = %v, want 0.4", p)
		}
	})

	t.Run("false camp only uses 1-F", func(t *testing.T) {
		p, err := probabilityTrue(0.4, false, 0.9, true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.1) {
			t.Errorf("P = %v, want 0.1", p)
		}
	})

	t.Run("coin-flip false camp leaves T unchanged", func(t *testing.T) {
		p, err := probabilityTrue(0.72, true, 0.5, true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.72) {
			t.Errorf("P = %v, want 0.72", p)
		}
	})

	t.Run("equal camps are 0.5", func(t *testing.T) {
		p, err := probabilityTrue(0.72, true, 0.72, true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.5) {
			t.Errorf("P = %v, want 0.5", p)
		}
	})

	t.Run("certain contradiction is 0.5", func(t *testing.T) {
		p, err := probabilityTrue(1, true, 1, true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.5) {
			t.Errorf("P = %v, want 0.5", p)
		}
	})

	t.Run("zero contradiction is 0.5", func(t *testing.T) {
		p, err := probabilityTrue(0, true, 0, true)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !almostEqual(p, 0.5) {
			t.Errorf("P = %v, want 0.5", p)
		}
	})
}

func TestSimpleTrustModelMetadata(t *testing.T) {
	model := NewSimpleTrustModel()
	if model.ID() != "simple" {
		t.Errorf("ID = %q, want simple", model.ID())
	}
	if model.Name() == "" {
		t.Error("Name should not be empty")
	}
	if model.Description() == "" {
		t.Error("Description should not be empty")
	}
}

func TestEvaluateBeforeSetup(t *testing.T) {
	d := newDebate(t)
	model := NewSimpleTrustModel()
	_, err := model.Evaluate(context.Background(), d.MoonIsRock)
	if !errors.Is(err, ErrNotSetup) {
		t.Errorf("Evaluate before Setup error = %v, want ErrNotSetup", err)
	}
}

func TestSetupNilResolver(t *testing.T) {
	model := NewSimpleTrustModel()
	err := model.Setup(context.Background(), nil, Roots{})
	if !errors.Is(err, ErrNilResolver) {
		t.Errorf("Setup(nil) error = %v, want ErrNilResolver", err)
	}
}

func TestIgnoredAssertionsUndefined(t *testing.T) {
	t.Run("Neutral's assertions are ignored", func(t *testing.T) {
		d := newDebate(t)
		model := d.setup(trust(d.Honest, 0.9))
		d.assertUndefined(model, d.MoonIsRock)
		d.assertUndefined(model, d.MoonIsCheese)
	})

	t.Run("Scammer is ignored unless trusted", func(t *testing.T) {
		d := newDebate(t)
		d.refute(d.Scammer, d.MoonIsRock, 0.6)
		d.affirm(d.Scammer, d.MoonIsCheese, 0.4)
		model := d.setup(trust(d.Honest, 0.9))
		d.assertUndefined(model, d.MoonIsRock)
		d.assertUndefined(model, d.MoonIsCheese)
	})

	t.Run("unknown assertion category is skipped", func(t *testing.T) {
		d := newDebate(t)
		d.claim(d.Honest.Unadorned(), d.MoonIsRock, assertions.Unknown, 1.0)
		d.assertUndefined(d.setup(trust(d.Honest, 0.9)), d.MoonIsRock)
	})

	t.Run("document refs are not assertions", func(t *testing.T) {
		d := newDebate(t)
		d.resolver.addRef(d.MoonIsRock, uri("gazette", "document"))
		d.assertUndefined(d.setup(trust(d.Honest, 0.9)), d.MoonIsRock)
	})

	t.Run("missing assertion is skipped", func(t *testing.T) {
		d := newDebate(t)
		d.resolver.addRef(d.MoonIsRock, uri("missing-a", "assertion"))
		d.assertUndefined(d.setup(trust(d.Honest, 0.9)), d.MoonIsRock)
	})

	t.Run("empty roots ignore Honest", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 1.0)
		d.assertUndefined(d.setup(), d.MoonIsRock)
	})
}

func TestUriAdornment(t *testing.T) {
	t.Run("adorned issuer still matches Honest", func(t *testing.T) {
		d := newDebate(t)
		d.claim(d.Honest.String(), d.MoonIsRock, assertions.IsTrue, 0.5)
		// Honest w=0.9×0.5=0.45; true camp only → P=0.45
		d.assertTrustScore(d.setup(trust(d.Honest, 0.9)), d.MoonIsRock, 0.45)
	})

	t.Run("adorned root key still matches Honest", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.8)
		// Honest w=0.7×0.8=0.56; true camp only → P=0.56
		d.assertTrustScore(d.setup(Roots{d.Honest.String(): 0.7}), d.MoonIsRock, 0.56)
	})
}

func TestRootValueClampedOnSetup(t *testing.T) {
	t.Run("trust above 1 is clamped", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 0.4)
		// trust 2.0 clamped to 1.0; w=1.0×0.4=0.4; true camp only → P=0.4
		d.assertTrustScore(d.setup(trust(d.Honest, 2.0)), d.MoonIsRock, 0.4)
	})

	t.Run("negative trust is clamped to 0", func(t *testing.T) {
		d := newDebate(t)
		d.affirm(d.Honest, d.MoonIsRock, 1.0)
		// trust -1.0 clamped to 0; w=0; true camp only → P=0
		d.assertTrustScore(d.setup(trust(d.Honest, -1.0)), d.MoonIsRock, 0)
	})
}

func TestFetchRefsError(t *testing.T) {
	d := newDebate(t)
	d.resolver.refsErr = errors.New("store unavailable")
	_, err := d.setup().Evaluate(context.Background(), d.MoonIsRock)
	if err == nil {
		t.Error("expected FetchRefs error")
	}
}

func TestSetupCopiesRoots(t *testing.T) {
	d := newDebate(t)
	d.affirm(d.Honest, d.MoonIsRock, 0.8)
	roots := trust(d.Honest, 0.5)
	model := d.setup(roots)
	roots[d.Honest.Unadorned()] = 1.0

	// Setup copied roots; mutating the map must not change w=0.5×0.8=0.4
	d.assertTrustScore(model, d.MoonIsRock, 0.4)
}
