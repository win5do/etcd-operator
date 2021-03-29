package controller

import (
	"strconv"

	errors2 "github.com/pkg/errors"
	"github.com/win5do/go-lib/errx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *controller) ListSvcNodePort(portName, namespace string, labels map[string]string) ([]int32, error) {
	found := &corev1.ServiceList{}
	if err := s.Kcli.ListByLabel(namespace, labels, found); err != nil {
		return nil, errx.WithStackOnce(err)
	}

	var nodePorts []int32

	for _, svc := range found.Items {
		for _, v := range svc.Spec.Ports {
			if v.Name != portName {
				continue
			}

			nodePorts = append(nodePorts, v.NodePort)
		}
	}

	if len(nodePorts) == 0 {
		return nil, errors2.New("invalid service")
	}

	return nodePorts, nil
}

func (s *controller) SyncSvc() error {
	cr := s.cr

	for i := 0; i < cr.Spec.Members; i++ {
		err := s.Kcli.EnsureService(s.Builder.ExportService(
			AddSuffix(cr.Name, Export, strconv.Itoa(i)),
			ExportSvcLabel(cr.ObjectMeta, i),
			MemberLabel(cr.ObjectMeta, i),
		))
		if err != nil {
			return errx.WithStackOnce(err)
		}
	}

	svcList := &corev1.ServiceList{}
	err := s.Kcli.ListByLabel(cr.Namespace, ExportSvcLabel(cr.ObjectMeta, SelectAll), svcList)
	if err != nil {
		return errx.WithStackOnce(err)
	}

	svcLen := len(svcList.Items)
	for svcLen > cr.Spec.Members {
		// service not support deletecollection. Ref: https://github.com/kubernetes/client-go/issues/505#issuecomment-440678666
		err := s.Kcli.DeleteObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AddSuffix(cr.Name, Export, strconv.Itoa(svcLen-1)),
				Namespace: cr.Namespace,
			},
		})
		if err != nil {
			return errx.WithStackOnce(err)
		}
		svcLen--
	}

	return nil
}
