package k8s

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/win5do/go-lib/logx"
	"k8s.io/client-go/rest"
)

const (
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"

	defaultNamespace = "etcd-operator-system"
)

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() string {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return defaultNamespace
		}
		log.Panicf("err: %+v", err)
	}
	ns := strings.TrimSpace(string(nsBytes))
	log.Debugf("Found namespace: %s", ns)
	return ns
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
