package trust

import (
	"context"
	"errors"
	"strings"

	"silvatek.uk/trustedassertions/internal/assertions"
	refs "silvatek.uk/trustedassertions/internal/references"
)

// ErrNotSetup is returned by Evaluate before Setup has been called on this instance.
var ErrNotSetup = errors.New("trust model has not been set up")

// ErrNilResolver is returned when Setup is called without a resolver.
var ErrNilResolver = errors.New("trust model requires a resolver")

// ErrUndefined is returned by Evaluate when no trusted IsTrue/IsFalse assertions contribute.
var ErrUndefined = errors.New("trust score is undefined")

// SimpleTrustModel is the v1 TrustModel. Same-direction claims combine as the
// probability they are not all wrong (T for IsTrue, F for IsFalse). Evaluate
// returns P(the statement is true), combining independent T and F; a missing
// camp is uninformative.
type SimpleTrustModel struct {
	resolver assertions.Resolver
	roots    Roots
	ready    bool
}

var _ TrustModel = (*SimpleTrustModel)(nil)

// NewSimpleTrustModel returns an unbound SimpleTrustModel. Call Setup before Evaluate.
func NewSimpleTrustModel() *SimpleTrustModel {
	return &SimpleTrustModel{}
}

func (m *SimpleTrustModel) ID() string {
	return "simple"
}

func (m *SimpleTrustModel) Name() string {
	return "Simple"
}

func (m *SimpleTrustModel) Description() string {
	return "Estimates P(statement is true) from trusted IsTrue and IsFalse assertions. Same-direction claims are the probability they are not all wrong; the two camps are combined as independent odds."
}

func (m *SimpleTrustModel) Setup(ctx context.Context, resolver assertions.Resolver, roots Roots) error {
	if resolver == nil {
		return ErrNilResolver
	}
	m.resolver = resolver
	m.roots = copyRoots(roots)
	m.ready = true
	return nil
}

func (m *SimpleTrustModel) Evaluate(ctx context.Context, statementUri refs.HashUri) (float64, error) {
	if !m.ready {
		return 0, ErrNotSetup
	}

	refs, err := m.resolver.FetchRefs(ctx, statementUri)
	if err != nil {
		return 0, err
	}

	trueWeights, falseWeights := m.calculateWeights(ctx, refs)

	hasTrue := len(trueWeights) > 0
	hasFalse := len(falseWeights) > 0
	t := 1 - probabilityAllWrong(trueWeights)
	f := 1 - probabilityAllWrong(falseWeights)
	return probabilityTrue(t, hasTrue, f, hasFalse)
}

func (m *SimpleTrustModel) calculateWeights(ctx context.Context, refs []refs.Reference) (trueWeights, falseWeights []float64) {
	for _, ref := range refs {
		if !isAssertion(ref) {
			continue
		}
		assertion, err := m.resolver.FetchAssertion(ctx, ref.Source)
		if err != nil {
			continue
		}
		weight, ok := m.weight(assertion)
		if !ok {
			continue
		}
		switch assertions.AssertionTypeOf(assertion.Category) {
		case assertions.IsTrue:
			trueWeights = append(trueWeights, weight)
		case assertions.IsFalse:
			falseWeights = append(falseWeights, weight)
		}
	}
	return trueWeights, falseWeights
}

func (m *SimpleTrustModel) weight(assertion assertions.Assertion) (float64, bool) {
	if assertion.RegisteredClaims == nil || assertion.Issuer == "" {
		return 0, false
	}

	issuerUri := refs.UriFromString(assertion.Issuer)
	rootTrust, ok := m.roots[issuerUri.Unadorned()]
	if !ok {
		return 0, false
	}

	switch assertions.AssertionTypeOf(assertion.Category) {
	case assertions.IsTrue, assertions.IsFalse:
		return clamp(rootTrust*float64(assertion.Confidence)), true
	default:
		return 0, false
	}
}

// probabilityTrue combines independent camp likelihoods T and F into
// P(statement is true). An absent camp is omitted, not treated as likelihood 0.
func probabilityTrue(t float64, hasTrue bool, f float64, hasFalse bool) (float64, error) {
	if !hasTrue && !hasFalse {
		return 0, ErrUndefined
	}
	if !hasFalse {
		return t, nil
	}
	if !hasTrue {
		return 1 - f, nil
	}
	denom := t*(1-f) + f*(1-t)
	if denom == 0 {
		return 0.5, nil
	}
	return t * (1 - f) / denom, nil
}

func probabilityAllWrong(weights []float64) float64 {
	allWrong := 1.0
	for _, w := range weights {
		allWrong *= 1 - clamp(w)
	}
	return allWrong
}

func isAssertion(ref refs.Reference) bool {
	return strings.EqualFold(ref.Source.Kind(), "assertion")
}
