# Trust models

This package estimates how likely a statement is to be true from assertions and a user's trust roots.

A `TrustModel` is bound once with `Setup(resolver, roots)`, then `Evaluate(statementUri)` returns `P(the statement is true)` in `[0, 1]`. Roots map unadorned entity URIs to trust levels in `[0, 1]`. Empty keys are skipped; levels are clamped.

If no trusted `IsTrue` or `IsFalse` assertions contribute, Evaluate returns `ErrUndefined` rather than a score. `Evaluate` before `Setup` returns `ErrNotSetup`. `Setup` with a nil resolver returns `ErrNilResolver`.

## SimpleTrustModel

`SimpleTrustModel` (`id: simple`) scores a statement from trusted `IsTrue` and `IsFalse` assertions whose issuer is already in the user's roots. Unknown categories, non-assertion refs, issuers not in roots, and failed assertion fetches are skipped.

Each contributing assertion has unsigned weight `w = clamp(rootTrust × confidence, 0, 1)`. Same-direction weights combine as the probability they are not all wrong (noisy-OR):

- `T = 1 − Π(1 − wᵢ)` over trusted IsTrue assertions
- `F = 1 − Π(1 − wⱼ)` over trusted IsFalse assertions

An absent camp is uninformative, not a likelihood of 0. The two camps are combined as independent odds under a uniform prior:

- No true and no false camp → undefined
- True camp only → `P = T`
- False camp only → `P = 1 − F`
- Both camps → `P = T(1 − F) / [T(1 − F) + F(1 − T)]`

If that denominator is 0 (`T = F = 0` or `T = F = 1`), the result is `0.5` (contradiction / no information).
