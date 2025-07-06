package artifact

import (
	"strings"
)

func ParseName(repoRef string, stripVersion bool) (string, error) {
	value := repoRef
	if i := strings.LastIndex(repoRef, "/"); i != -1 {
		value = repoRef[i+1:]
	}

	if stripVersion {
		return TrimVersion(value), nil
	}
	return value, nil
}

func TrimVersion(v string) string {
	if name, _, found := strings.Cut(v, ":"); found {
		return name
	}
	return v
}
