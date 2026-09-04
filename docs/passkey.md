# Passkey support

Passkeys (WebAuthn / FIDO2) sit alongside bcrypt passwords. They authenticate the *user account*. They do not replace entity certificates or JWT assertions.

Identity today is a bcrypt hash on `auth.User` (`PassHash`) and a JWT in the `auth` cookie (`SetAuthCookie`, 2 hour lifetime, `HttpOnly`, `SameSite=Strict`, `Secure` on HTTPS). Registration is invite-code plus password.

Server library: `github.com/go-webauthn/webauthn`. Browser: native `navigator.credentials` in `web/static/passkey.js`. Do not serve the ceremony through HTMX HTML swaps.

An old investigation lives on the `passkeys` branch (hardcoded user, begin-registration only). Production work is on `main` via PRs #5–#9 plus the login form / `passkey.js` ceremony.

## Goals

- Existing users can add passkeys while logged in. **Done** (profile page).
- Sign in with a passkey, then issue the same JWT cookie as password login. **Done** (PR #9 + login UI).
- Keep passwords. Passkeys are an additional authenticator, not a cutover.
- Work on localhost and on Cloud Run behind HTTPS, including multiple instances (ceremony session is an encrypted cookie, not process memory).

Non-goals for a first version: passwordless-only accounts, usernameless / Conditional UI, cross-device QR as a special flow, using the passkey as a *signing* key for assertions.

## Done on main

### Config + wrapper (PR #5)

- `WEBAUTHN_RP_ORIGIN` — full origin the browser sees (`http://127.0.0.1:8080` or `https://trustedassertions.silvatek.uk`).
- `WEBAUTHN_RP_NAME` — display name, `Trusted Assertions`.
- RP ID is the origin hostname (no scheme, path, or port). Do not hardcode `localhost`.
- `WEBAUTHN_RP_ORIGIN` is set on Cloud Run to the public hostname.
- `auth.User` implements `webauthn.User`.
- Ceremony session from `BeginRegistration` / `BeginLogin` is an encrypted, `HttpOnly`, `SameSite=Strict` cookie (`webauthn_session`). CSRF key is reused as the cookie secret.

### Persistence (PRs #6–#7, login use in PR #9)

- `auth.User.Passkeys []Passkey`, cap `auth.MaxPasskeys` (5).
- `Passkey` holds credential ID, public key, attestation/flags, sign count, transports, AAGUID, attachment, `Name`, `CreatedAt`, `LastUsedAt`.
- `PasskeyFromCredential` / `Credential()` round-trip to `webauthn.Credential`. `CreatedAt` is set in `PasskeyFromCredential`.
- Datastore helpers on the controller: `AddPasskey`, `RemovePasskey` (`RemovePasskey` has no UI yet), `RecordPasskeyUse` (sign count, flags, `CloneWarning`, `LastUsedAt`; leaves `Name` and `CreatedAt` alone).
- In-memory `FetchUser` copies the passkeys slice; nested `[]byte` still aliases (known stopgap).

### Register while logged in (PR #8)

JSON under `/web/passkey/…`, CSRF via `X-CSRF-Token`. User verification required.

| Step | Method | Auth | Status |
| --- | --- | --- | --- |
| Register begin | POST `/web/passkey/register/begin` | logged in | Done. `excludeCredentials` for existing IDs. 409 at cap. |
| Register finish | POST `/web/passkey/register/finish` | logged in | Done. `FinishRegistration` then `datastore.AddPasskey`. |
| Login begin | POST `/web/passkey/login/begin` | anonymous | Done. Identifier-first `{ "user_id": "..." }`, `BeginLogin`, `UserVerification: required`. Unknown user, empty ID, or no passkeys: 401 `"Unable to verify identity"`. Unconfigured WebAuthn: 503. |
| Login finish | POST `/web/passkey/login/finish` | anonymous | Done. `FinishLogin`, then `datastore.RecordPasskeyUse`. `CloneWarning` rejects without an `auth` cookie (warning already persisted). Otherwise `SetAuthCookie` and `{ "redirect": "/web/home" }`. |
| List | GET `/web/profile` | logged in | Done (table on the profile page). |
| Revoke | POST | logged in | **Not done.** Endpoint was implemented then removed; bring back later. |

Display names are derived at registration (`derivedPasskeyName`): known AAGUID map, else USB/NFC/BLE → “Security key”, hybrid → “Phone”, platform + backup flags → “Synced passkey”, platform → “This device”, else “Passkey”. `Name` is still persisted so user-editable nicknames can return later.

Profile (`web/viewprofile.html`): table of name / added / last used, “Add passkey” (hidden if `PublicKeyCredential` is missing). No name field.

### Login (PR #9 + login UI)

Identifier-first: the user types their user ID, then password or passkey. Matches the current login form and avoids `FindUserByCredentialID` until usernameless login is needed.

Login page (`web/loginform.html`): password form kept (`hx-boost="false"`). “Use a passkey” (`#use-passkey`, `hx-boost="false"`) and `#passkey-status`; button hidden when `PublicKeyCredential` is missing. `passkey.js` `taPasskeys.login` POSTs begin, runs `navigator.credentials.get`, POSTs finish with `X-CSRF-Token`, then `location.assign` the returned `redirect`.

Tests: `internal/auth` name/round-trip tests; `internal/web/passkey_web_test.go` for profile UI and begin/finish auth. Real authenticator UI is not exercised in `httptest`.

Passkey registration has been tested on https://trustedassertions.silvatek.uk/ with user
admin@trustedassertions.silvatek.uk and a Google Password Manager passkey. Attempting to add a second passkey
for the same user and browser results in the expected error message 
"The user attempted to register an authenticator that contains one of the credentials already registered with the relying party."

## Remaining

### Later

- Revoke on the profile page (`RemovePasskey`). Revoking must not log other sessions out; the JWT is self-contained.
- User-editable passkey names.
- Discoverable (usernameless) credentials and Conditional UI.
- Extra tests around verify/store with fixtures. Skip real authenticator UI in `httptest`.
- Register a passkey while registering a user (no password).

## Security notes

- HTTPS in production; `localhost` / `127.0.0.1` are the HTTP origins for local work.
- CSRF on finish endpoints; attestation/assertion JSON is not a substitute.
- Update and check the signature counter on every login.
- Same auth cookie properties as password login.
