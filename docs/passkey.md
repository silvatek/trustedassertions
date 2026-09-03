# Adding passkey support

This note describes how passkeys (WebAuthn / FIDO2) would be added to Trusted Assertions without replacing the existing password login.

Today, identity is a bcrypt password on `auth.User` (`PassHash`) and a JWT in the `auth` cookie (`SetAuthCookie`, 2 hour lifetime, `SameSite=Strict`). Registration is invite-code plus password. The profile page only lists signing keys. Login and register already set `hx-boost="false"` because they are full document POSTs with CSRF.

A previous investigation on the `passkeys` branch used `github.com/go-webauthn/webauthn` and `@simplewebauthn/browser`. It got as far as beginning registration for a hardcoded user on `localhost`. It did not persist credentials, store ceremony session data, finish verification, or work off localhost. That is still the right stack; the rest of this plan is the production-shaped version of that POC.

## Goals

- Let an existing user add one or more passkeys while logged in.
- Let them sign in with a passkey, then issue the same JWT cookie as password login.
- Keep passwords. Passkeys are an additional authenticator, not a cutover.
- Work on localhost and on Cloud Run behind HTTPS, including multiple instances (no in-process session store).

Non-goals for a first version: passwordless-only accounts, cross-device QR pairing as a special flow (the platform handles that), and using the passkey as a *signing* key for assertions. Assertion signing stays on the existing entity private keys.

## Data model

Extend `auth.User` (or a sibling document keyed by user ID) with a list of WebAuthn credentials:

- credential ID
- public key
- attestation type / flags
- signature counter (clone detection)
- transports
- friendly name, created-at, last-used-at

`go-webauthn` expects a `webauthn.User` implementation. Wrap `auth.User` rather than a separate dummy type. Firestore already stores the whole `User` document (`UserCollection`). Binary fields should be stored as bytes or base64 so `StoreUser` / `FetchUser` do not need a new collection at first.

Add datastore helpers:

- `AddPasskey(userID, credential)`
- `RemovePasskey(userID, credentialID)`
- `FindUserByCredentialID` only if you later add usernameless login

Cap the number of passkeys per user (for example 5).

## Configuration

WebAuthn is origin-bound. Do not hardcode `localhost`.

Environment (or equivalent):

- `WEBAUTHN_RP_ORIGIN` — full origin, e.g. `http://127.0.0.1:8080` or `https://trustedassertions.silvatek.uk`
- `WEBAUTHN_RP_NAME` — display name, `Trusted Assertions`

RP ID is the origin hostname (no scheme, path, or port). The origin must be the host the browser sees (not Cloud Run’s internal URL). The load balancer hostname is the RP ID in production.

Ceremony session data from `BeginRegistration` / `BeginLogin` must survive until `Finish*`. Cloud Run is stateless and scales to zero, so keep that session in an encrypted, `HttpOnly`, `SameSite=Strict` cookie (or a short-lived Firestore doc). Do not keep it only in process memory.

## HTTP API

JSON endpoints under `/web/passkey/…` (same CSRF cookie as the rest of `/web`). The login form already demonstrates sending CSRF on a non-HTMX POST; JSON callers should send `X-CSRF-Token`.

| Step | Method | Auth | Purpose |
| --- | --- | --- | --- |
| Register begin | POST | logged in | `webAuthn.BeginRegistration`; return `publicKey` options |
| Register finish | POST | logged in | `FinishRegistration`; store credential |
| Login begin | POST | anonymous | `BeginLogin` for the given user ID |
| Login finish | POST | anonymous | `FinishLogin`; `SetAuthCookie` |
| List / delete | GET / POST | logged in | manage passkeys on the profile |

Require **user verification** (PIN / biometrics) on both register and login.

Login is identifier-first: the user still types their user ID, then chooses password or passkey. That matches the current login form and avoids credential-lookup-by-ID until it is needed.

After a successful passkey login, call the existing `SetAuthCookie` so the rest of the app is unchanged.

## UI

**Login** (`web/loginform.html`): keep the password form. Add a “Use a passkey” control that runs only if `window.PublicKeyCredential` exists. Small dedicated JS; do not HTMX-boost these calls (same reason the password form already disables boost). Optional later: WebAuthn Conditional UI (autofill) on the user ID field.

**Profile** (`web/viewprofile.html`): list passkeys with name and last used, “Add passkey”, revoke. Registration of a passkey happens here after password (or existing passkey) login.

**Register**: leave as password + invite code. After first login, the profile can prompt to add a passkey. Adding passkeys during invite registration can wait.

## Security notes

- HTTPS in production; `localhost` is the only HTTP origin.
- CSRF on finish endpoints; the attestation/assertion JSON is not a substitute for CSRF.
- Update and check the signature counter on every login.
- Same auth cookie properties as today (`Path=/`, `SameSite=Strict`). Consider `Secure` when not on localhost (the cookie is not `Secure` today).
- Generic errors on login failure (same as `ErrorAuthFail`) so passkey vs password is not leaked more than the current user-not-found path.
- Revoking a passkey must not log other sessions out unless you later add server-side session revocation; the JWT cookie is currently self-contained.

Passkeys authenticate the *user account*. They do not replace entity certificates or JWT assertions in the trust model.

## Implementation order

1. **Config + wrapper** — initialise `webauthn.WebAuthn` from env; implement `webauthn.User` on the real user; encrypted ceremony cookie.
2. **Persistence** — credential fields on `User`; memory store and Firestore round-trip tests.
3. **Register while logged in** — begin/finish + profile UI + revoke.
4. **Login** — begin/finish + login page button; reuse `SetAuthCookie`.
5. **Tests** — unit tests around verify/store with fixtures; web tests for profile list and “passkey button present”; skip real authenticator UI in `httptest`.
6. **Deploy** — set RP ID/origin on Cloud Run to the public hostname; confirm the load balancer origin matches.

A follow-up could add discoverable (usernameless) credentials and Conditional UI. The auth cookie is already `HttpOnly` and `SameSite=Strict`, and `Secure` on HTTPS (including `X-Forwarded-Proto`).

## Libraries

- Server: `github.com/go-webauthn/webauthn` (already used in the POC).
- Browser: either `@simplewebauthn/browser` (POC) or the native `navigator.credentials` API. Prefer a small local script on login/profile rather than a standalone `passkey.html` page.

Do not serve the WebAuthn ceremony through HTMX HTML swaps; the browser API and the JSON ceremony are a better fit for `fetch`.
