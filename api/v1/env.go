package v1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	DEBUG = "DEBUG"
	Empty = "EMPTY" // crd有bug，空的env会被填上值，使用empty占位符
)

func GetEnv(env []corev1.EnvVar, key string) string {
	r, _ := LookupEnv(env, key)
	return r
}

func LookupEnv(env []corev1.EnvVar, key string) (string, bool) {
	for _, e := range env {
		if e.Name == key {
			return e.Value, true
		}
	}
	return "", false
}

func SetEnv(env []corev1.EnvVar, key, val string) []corev1.EnvVar {
	var found bool
	for i := range env {
		if env[i].Name == key {
			found = true
			env[i].Value = val
			break
		}
	}

	if !found {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: val,
		})
	}

	return env
}

func SetEnvIfUnset(env []corev1.EnvVar, key, def string) []corev1.EnvVar {
	val, ok := LookupEnv(env, key)
	if ok {
		if val == "" {
			return SetEnv(env, key, Empty)
		}
		return env
	}

	if def == "" {
		def = Empty
	}

	return SetEnv(env, key, def)
}

func MergeEnv(src, dst []corev1.EnvVar) []corev1.EnvVar {
	//  使用set会去重
	set := make(map[string]string)
	for _, e := range src {
		set[e.Name] = e.Value
	}

	for _, e := range dst {
		_, ok := set[e.Name]
		if !ok {
			set[e.Name] = e.Value
		}
	}

	var r []corev1.EnvVar
	for k, v := range set {
		r = append(r, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return r
}
