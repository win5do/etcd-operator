package conf

import (
	"os"
	"strings"

	errors2 "github.com/pkg/errors"
	log "github.com/win5do/go-lib/logx"
	corev1 "k8s.io/api/core/v1"
)

var globalConfig Config

func GetGlobalConfig() Config {
	return globalConfig
}

func init() {
	globalConfig = configFromEnv()
}

type Config struct {
	IMAGE              string
	STORAGE_CLASS_NAME string
	EXTERNAL_DOMAIN    string
	INSTANCE_ENV       string // 实例env，多个键值对，`;`分隔
	InstanceEnv        []corev1.EnvVar
}

func configFromEnv() Config {
	c := Config{
		IMAGE:              getEnv("IMAGE", "bitnami/etcd:3"),
		STORAGE_CLASS_NAME: getEnv("STORAGE_CLASS_NAME", ""),
		EXTERNAL_DOMAIN:    getEnv("EXTERNAL_DOMAIN", "gogo.io"),
		INSTANCE_ENV:       getEnv("INSTANCE_ENV", ""),
	}

	kvs, err := parseKV(c.INSTANCE_ENV)
	if err != nil {
		log.Panic(err)
	}
	c.InstanceEnv = func(kvs [][]string) []corev1.EnvVar {
		var r []corev1.EnvVar
		for _, kv := range kvs {
			r = append(r, corev1.EnvVar{
				Name:  kv[0],
				Value: kv[1],
			})
		}
		return r
	}(kvs)

	return c
}

func getEnv(name, defaultVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return val
}

func parseKV(s string) ([][]string, error) {
	kvarr := strings.Split(s, ";")
	var r [][]string

	for _, kvstr := range kvarr {
		if kvstr == "" {
			continue
		}

		kv := strings.SplitN(kvstr, "=", 2)
		if len(kv) < 2 || kv[1] == "" {
			return nil, errors2.Errorf("invalid kv: %s", kvstr)
		}
		r = append(r, kv)
	}

	return r, nil
}
