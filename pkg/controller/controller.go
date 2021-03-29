package controller

import (
	"go.uber.org/zap"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
	"github.com/win5do/etcd-operator/pkg/conf"
	"github.com/win5do/etcd-operator/pkg/k8s"
)

type controller struct {
	reqLog *zap.SugaredLogger
	cr     *dbv1.Etcd
	cfg    conf.Config

	Kcli          *k8s.Kcli
	Builder       *ResourceBuilder
	StatusManager *statusManager
}
