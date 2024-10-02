package assertions

import "strings"

type HashUri string

func MakeUri(hash string, kind string) HashUri {
	uri := "hash://sha256/" + hash
	if kind != "" {
		uri = uri + "?type=" + kind
	}
	return HashUri(uri)
}

func (u *HashUri) Hash() string {
	hash := strings.TrimPrefix(u.string(), "hash://sha256/")
	index := strings.Index(hash, "?type=")
	if index > -1 {
		hash = hash[:index]
	}
	return hash
}

func (u *HashUri) Alg() string {
	return "sha256"
}

func (u *HashUri) string() string {
	return string(*u)
}

func (u *HashUri) Short() string {
	s := u.Hash()
	if len(s) > 8 {
		return s[len(s)-8:]
	} else {
		return s
	}
}

func (u *HashUri) Kind() string {
	hash := u.Hash()
	index := strings.Index(hash, "?type=")
	if index == -1 {
		return "unknown"
	} else {
		return hash[index+6:]
	}
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

func (u *HashUri) WebPath() string {
	return "/web/" + mapPathType(u.Kind()) + "/" + u.Hash()
}

func (u *HashUri) ApiPath() string {
	return "/api/v1/" + mapPathType(u.Kind()) + "/" + u.Hash()
}
