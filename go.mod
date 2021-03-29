module github.com/win5do/etcd-operator

go 1.15

require (
	github.com/go-logr/zapr v0.2.0
	github.com/google/wire v0.5.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/open-policy-agent/cert-controller v0.1.1-0.20210308205344-203624759536
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/win5do/go-lib v0.0.0-20210322065409-edc6813f5414
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
)

replace github.com/open-policy-agent/cert-controller => ./pkg/lib/cert-controller
