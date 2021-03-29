package test

import (
	log "github.com/win5do/go-lib/logx"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
)

func Kconf() *rest.Config {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("cluster url: %s", cfg.Host)
	return cfg
}

func Kscheme() *runtime.Scheme {
	scheme, err := dbv1.SchemeBuilder.Build()
	if err != nil {
		log.Panic(err)
	}
	return scheme
}

func Kcli(sbs ...runtime.SchemeBuilder) client.Client {
	cfg := Kconf()

	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		log.Panic(err)
	}

	scheme := kubescheme.Scheme
	sbs = append(
		sbs,
		dbv1.SchemeBuilder.SchemeBuilder,
	)

	for _, sb := range sbs {
		err := sb.AddToScheme(scheme)
		if err != nil {
			log.Panic(err)
		}
	}

	cli, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		log.Panic(err)
	}

	return cli
}
