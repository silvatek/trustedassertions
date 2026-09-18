package trust

import (
	"context"

	"silvatek.uk/trustedassertions/internal/assertions"
	refs "silvatek.uk/trustedassertions/internal/references"
)

// Roots maps unadorned entity URIs (hash://sha256/{hex}) to trust levels in [0, 1].
type Roots map[string]float64

// TrustModel scores statements against a bound resolver and trust roots.
// Call Setup once, then Evaluate any number of statement URIs.
// Evaluate returns P(the statement is true) in [0, 1], or an error when undefined.
type TrustModel interface {
	ID() string
	Name() string
	Description() string
	Setup(ctx context.Context, resolver assertions.Resolver, roots Roots) error
	Evaluate(ctx context.Context, statementUri refs.HashUri) (float64, error)
}

// GetModel returns a fresh TrustModel for id, already bound with Setup.
// Empty or unknown id uses "simple".
func GetModel(ctx context.Context, id string, roots Roots, resolver assertions.Resolver) (TrustModel, error) {
	var model TrustModel
	switch id {
	default:
		model = &SimpleTrustModel{}
	}
	if err := model.Setup(ctx, resolver, roots); err != nil {
		return nil, err
	}
	return model, nil
}

func copyRoots(in Roots) Roots {
	out := make(Roots, len(in))
	for k, v := range in {
		if k == "" {
			continue
		}
		out[refs.UriFromString(k).Unadorned()] = clamp(v)
	}
	return out
}

func clamp(v float64) float64 {
	return clampTo(v, 0, 1)
}

func clampTo(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
