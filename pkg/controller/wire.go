//go:generate wire
// +build wireinject

package controller

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
	"github.com/win5do/etcd-operator/pkg/conf"
	"github.com/win5do/etcd-operator/pkg/k8s"
)

func Inject(cli client.Client, scheme *runtime.Scheme, cr *dbv1.Etcd, log *zap.SugaredLogger, cfg conf.Config) *controller {
	wire.Build(
		wire.Bind(new(metav1.Object), new(*dbv1.Etcd)),
		k8s.NewKcli,
		NewResourceBuilder,
		wire.Struct(new(statusManager), "*"),
		wire.Struct(new(controller), "*"),
	)
	return nil
}
