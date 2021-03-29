package conf

import (
	"testing"
)

func TestHostAlias(t *testing.T) {
	in := []string{
		"a=1",
		"a=1;b=2",
		"a=1;b=2;",
		"a=",
		"",
	}

	for _, v := range in {
		r, err := parseKV(v)
		t.Log(r, err)
	}
}
