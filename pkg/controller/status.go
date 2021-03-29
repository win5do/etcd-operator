package controller

import (
	"fmt"

	"github.com/win5do/go-lib/errx"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
	"github.com/win5do/etcd-operator/pkg/k8s"
	"github.com/win5do/etcd-operator/pkg/rerr"
)

type statusManager struct {
	kcli *k8s.Kcli
}

func (s *statusManager) CheckDeployReady(meta metav1.ObjectMeta) (dbv1.NodeStatus, error) {
	list := &appsv1.StatefulSetList{}
	err := s.kcli.ListByLabel(meta.Namespace, MemberLabel(meta, SelectAll), list)
	if err != nil {
		return dbv1.StatusUnknown, errx.WithStackOnce(err)
	}

	var expectReady int32 = 0
	var zkReady int32 = 0

	for _, v := range list.Items {
		expectReady += v.Status.Replicas

		zkReady += v.Status.ReadyReplicas
	}

	var status dbv1.NodeStatus

	if zkReady == expectReady {
		status = dbv1.StatusReady
	} else if zkReady == 0 {
		status = dbv1.StatusFailed
	} else {
		status = dbv1.StatusPartialReady
	}

	return status, nil
}

func (s *statusManager) UpdateStatus(cr *dbv1.Etcd, status dbv1.EtcdStatus) error {
	cr.Status = status

	return s.kcli.WriteStatus(cr)
}

func (s *statusManager) HandleStatus(cr *dbv1.Etcd, nodePorts []int32) error {
	status, err := s.CheckDeployReady(cr.ObjectMeta)
	if err != nil {
		return errx.WithStackOnce(err)
	}

	err = s.UpdateStatus(cr, dbv1.EtcdStatus{
		Status:      status,
		ConnectAddr: externalConnectAddr(cr.Spec.ExternalHost, nodePorts),
	})
	if err != nil {
		return errx.WithStackOnce(err)
	}

	if status != dbv1.StatusReady {
		return rerr.Err_wait_requeue
	}

	return nil
}

func externalConnectAddr(host string, nodePorts []int32) string {
	r := ""

	f := "%s:%d,"

	for _, port := range nodePorts {
		r += fmt.Sprintf(f, host, port)
	}

	return r[:len(r)-1] // rm trailing comma
}
