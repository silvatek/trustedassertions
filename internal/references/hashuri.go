package references

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
)

type HashUri struct {
	uri string
}

var EMPTY_URI = HashUri{uri: ""}
var ERROR_URI = HashUri{uri: "ERROR"}
var TYPE_QUERY = "?type="

func MakeUri(hash string, kind string) HashUri {
	uri := "hash://sha256/" + hash
	if kind != "" {
		uri = uri + TYPE_QUERY + strings.ToLower(kind)
	}
	return HashUri{uri: uri}
}

func MakeUriB(hash []byte, kind string) HashUri {
	return MakeUri(fmt.Sprintf("%x", hash), kind)
}

func UriFor(ref Referenceable) HashUri {
	return UriFromContent(ref.Content(), ref.Type())
}

// Create a HashUri from a string.
//
// The string can be a hash, a raw URI or an escaped URI.
func UriFromString(str string) HashUri {
	if strings.HasPrefix(str, "hash://sha256/") {
		// Is a raw URI string
		return HashUri{uri: str}
	}
	if strings.HasPrefix(str, "hash:%2F%2Fsha256%2F") {
		// It is an escaped URI string
		return UnescapeUri(str, "")
	}
	return MakeUri(str, "") // Assume it is just a hash
}

// Create a HashUri from content.
//
// The content is hashed using the default algorithm and the HashUri built from that hash.
func UriFromContent(content string, kind string) HashUri {
	hash := sha256.New()
	hash.Write([]byte(content))
	return MakeUriB(hash.Sum(nil), kind)
}

func (u HashUri) Hash() string {
	hash := strings.TrimPrefix(u.String(), "hash://sha256/")
	index := strings.Index(hash, TYPE_QUERY)
	if index > -1 {
		hash = hash[:index]
	}
	return hash
}

func (u HashUri) Alg() string {
	return "sha256"
}

func (u HashUri) String() string {
	return u.uri
}

func (u HashUri) Short() string {
	s := u.Hash()
	if len(s) > 8 {
		return s[len(s)-8:]
	} else {
		return s
	}
}

func kind(uri string) string {
	index := strings.Index(uri, TYPE_QUERY)
	if index == -1 {
		return "unknown"
	} else {
		return uri[index+6:]
	}
}

func (u HashUri) Kind() string {
	return kind(u.uri)
}

func (u HashUri) HasType() bool {
	return strings.Contains(u.uri, TYPE_QUERY)
}

func mapPathType(kind string) string {
	kind = strings.ToLower(kind)
	if kind == "unknown" || kind == "" {
		return "error"
	} else if kind == "entity" {
		return "entities"
	} else {
		return kind + "s"
	}
}

func (u HashUri) Unadorned() string {
	s := u.String()
	index := strings.Index(s, TYPE_QUERY)
	if index > -1 {
		return s[0:index]
	}
	return s
}

// Returns a URL path-escaped version of the unadorned URI.
//
// This can be used as the key in a key/pair storage model, or in HTTP requests.
func (u HashUri) Escaped() string {
	return url.PathEscape(u.Unadorned())
}

func UnescapeUri(uri string, kind string) HashUri {
	s, err := url.PathUnescape(uri)
	if err != nil {
		return EMPTY_URI
	}
	if kind != "" {
		s += TYPE_QUERY + kind
	}
	return HashUri{uri: s}
}

func (u HashUri) WebPath() string {
	return "/web/" + mapPathType(u.Kind()) + "/" + u.Hash()
}

func (u HashUri) ApiPath() string {
	return "/api/v1/" + mapPathType(u.Kind()) + "/" + u.Hash()
}

func (u HashUri) IsEmpty() bool {
	return u.uri == ""
}

func (u HashUri) Len() int {
	return len(u.uri)
}

func (u HashUri) WithType(kind string) HashUri {
	u2 := HashUri{uri: u.uri + TYPE_QUERY + strings.ToLower(kind)}
	return u2
}

func (u HashUri) Equals(other HashUri) bool {
	return u.Escaped() == other.Escaped()
}
