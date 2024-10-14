# Trusted Assertions
The ***Trusted Assertions Framework*** provides a mechanism for statements and assertions about those statements to be recorded and used as the basis for a trust model.

## Data Model
The core data model consists of three main data types:-

* A `Statement` is some text, identified by a hash of its content, about which assertions can be made by entities.
* An `Entity` is an X509 certificate, identified by its serial number, representing an individual or organisation.
* An `Assertion` is a JSON Web Token, identified by its signature, containing claims made by an Entity about a Statement or another Assertion.

In addition to these primary data types, there is also the `Reference` which is a combination of a target, a source and a reference type. References are identified from within assertions and stored separately as a form of index.

For servers that will be creating new Entities or Assertions, it will also be necessary to store (or at least have access to) the private keys for the signing entities.

The data in the data model is immutable; data is only added, never modified. It is content-addressable; the data should be stored in its plain-text representation, and accessed using a URI built from a hash of that text.


### Data URIs
URIs for statements, entities and assertions are based on a digital hash of the content. The content for statements is the text, for entities it is the X509 certificate text, and for assertions it is the JWT text.

See https://github.com/hash-uri/hash-uri

E.g.  `hash://sha256/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`


### Assertion Claims

#### Registered claims

* `aud` (audience) should be a single value, which is the Trusted Assertions general audience (including version), which is currently "trustedassertions:0.1/any"
* `iss` (issuer) should be the URI of the `Entity` making the assertion
* `sub` (subject) should be the URI of the `Statement`, `Entity` or `Assertion` that this assertion is about

#### Custom Claims

* `category` is the type of claim made by the assertions, such as...
    * `IsTrue`
    * `IsFalse`
    * `Replaces`
    * `IsSameAs`
    * `IsCompromised`
* `object` is the URI of the object of the claim, for assertions that relate multiple URIs, such as "Replaces"
* `confidence` is the confidence of the claim, from 0.0 (no conficence) to 1.0 (fully confident)
* `basis` as a list of URIs of other assertions that support this assertion

## Trust Models

A trust model is a mechanism for estimating how likely any individual statement is to be true, by following chains of assertions back to entities.

The root of a trust model is a set of entities that the user of the model has some level of trust in. Different users can supply different sets of trusted entities to the same trust model, and will get different outcomes from the model.

Any number of trust models can be created from the same set of assertions, and it is anticipated that the science of trust modelling will evolve significantly over time.

## Development Commands

* `go run ./cmd/server/main.go`
* `go test -coverprofile=coverage.out ./...`
* `go tool cover -html=coverage.out`

## Things to Do

* QR Codes for statement, entity and assertion pages 
* SubjectType and ObjectType claims in assertions, or auto-detect type
* Web page for adding assertion to existing statements etc
* Mobile web views
* Web page to create new entity
* Secure management of private keys
* Access control
* Logout
* User JWT refresh
* User management

### Done

* Initial framework
* Minimal API
* Firestore data store
* Structured logging
* Initial web UI
* Web UI to create statement & assertion
* Error handling
* Basic stylesheets
* User authentication