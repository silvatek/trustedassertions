# Trusted Assertions
The Trusted Assertions framework provides a mechanism for statements and assertions about those statements to be recorded and used as the basis for a trust model.

## Data Model
The core data model consists of three main data types:-

* A `Statement` is some text, identified by a hash of its content, about which assertions can be made by entities.
* An `Entity` is an X509 certificate, identified by its serial number, representing an individual or organisation.
* An `Assertion` is a JSON Web Token, identified by its signature, containing claims made by an Entity about a Statement or another Assertion.

In addition to these primary data types, there is also the `Reference` which is a combination of a target, a source and a reference type. References are identified from within assertions and stored separately as a form of index.

### Statement URI
See https://github.com/hash-uri/hash-uri

E.g.  `hash://sha256/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

### Entity URI

E.g. `cert://x509/846759547388737982927`

### Assertion URI

E.g. `sig://jwt/eyJhbGciOiJSUzI1NiIsImtpbmQiOiJFbnRpdHkiLCJ0eXAiOiJKV1QifQ`

