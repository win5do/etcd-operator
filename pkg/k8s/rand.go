package k8s

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"reflect"

	log "github.com/win5do/go-lib/logx"
	randk8s "k8s.io/apimachinery/pkg/util/rand"
)

func getStringFieldVal(v interface{}, field string) string {
	r := reflect.ValueOf(v)
	rv := reflect.Indirect(r)

	if rv.Kind() != reflect.Struct {
		// 必须为结构体
		return ""
	}

	f := rv.FieldByName(field)
	if f.IsValid() {
		return f.String()
	} else {
		return ""
	}
}

const (
	RANDOM_SUFFIX_LEN = 10
	// k8s object name has a maximum length
	MAX_NAME_LEN = 63 - RANDOM_SUFFIX_LEN - 1
)

func GenerateKey(ln int) []byte {
	b := make([]byte, ln)
	_, err := rand.Read(b)
	if err != nil {
		log.Error(err, "GenerateKey err")
		return nil
	}
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(buf, b)
	return buf
}

func AddRandSuffix(name string) string {
	suffix := randk8s.String(RANDOM_SUFFIX_LEN)
	if len(name) > MAX_NAME_LEN {
		log.Warn("name too long, cutoff it")
		name = name[:MAX_NAME_LEN]
	}

	return fmt.Sprintf("%s-%s", name, suffix)
}
