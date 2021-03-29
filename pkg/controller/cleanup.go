package controller

import (
	"github.com/win5do/go-lib/errx"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const etcdFinalizer = "etcd-operator/finalizer"

func (s *controller) Finalize() error {
	if contains(s.cr.GetFinalizers(), etcdFinalizer) {
		// Run finalization logic for memcachedFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		err := s.cleanup()
		if err != nil {
			return errx.WithStackOnce(err)
		}

		err = s.removeFinalizer()
		if err != nil {
			return errx.WithStackOnce(err)
		}

		s.reqLog.Info("success finalized")
	}

	return nil
}

func (s *controller) cleanup() error {
	// clean PVC
	err := s.Kcli.DeleteALLByLabel(&corev1.PersistentVolumeClaim{}, s.cr.Namespace,
		MemberLabel(s.cr.ObjectMeta, SelectAll))
	if err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *controller) AddFinalizer() error {
	cr := s.cr

	if !contains(cr.GetFinalizers(), etcdFinalizer) {
		controllerutil.AddFinalizer(cr, etcdFinalizer)
		err := s.Kcli.UpdateObject(cr)
		if err != nil {
			return errx.WithStackOnce(err)
		}
	}

	return nil
}

func (s *controller) removeFinalizer() error {
	cr := s.cr

	// Remove memcachedFinalizer. Once all finalizers have been
	// removed, the object will be deleted.
	controllerutil.RemoveFinalizer(cr, etcdFinalizer)
	err := s.Kcli.UpdateObject(cr)
	if err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
