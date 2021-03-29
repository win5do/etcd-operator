package k8s

import (
	"context"
	"encoding/json"

	errors2 "github.com/pkg/errors"
	"github.com/win5do/go-lib/errx"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlcli "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/win5do/etcd-operator/pkg/rerr"
)

type Kcli struct {
	client ctrlcli.Client
	scheme *runtime.Scheme
	log    *zap.SugaredLogger
	owner  metav1.Object
}

func NewKcli(client ctrlcli.Client, scheme *runtime.Scheme, log *zap.SugaredLogger, obj metav1.Object) *Kcli {
	return &Kcli{
		client: client,
		scheme: scheme,
		log:    log,
		owner:  obj,
	}
}

func (s *Kcli) Find(name, namespace string, found ctrlcli.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	return s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, found)
}

func (s *Kcli) ListByLabel(namespace string, labels map[string]string, found ctrlcli.ObjectList) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	err := s.client.List(ctx, found, ctrlcli.InNamespace(namespace), ctrlcli.MatchingLabels(labels))
	return errx.WithStackOnce(err)
}

func (s *Kcli) IsExists(obj metav1.Object, found ctrlcli.Object) (exists bool, err error) {
	err = s.Find(obj.GetName(), obj.GetNamespace(), found)
	if err == nil {
		return true, nil
	}
	if !k8serr.IsNotFound(err) {
		return false, errors2.Wrap(err, "not found")
	}

	return false, nil
}

func (s *Kcli) SetRefAndCreateObject(obj interface{}) error {
	// Set cr as the owner and controller
	err := controllerutil.SetControllerReference(s.owner, obj.(metav1.Object), s.scheme)
	if err != nil {
		return errx.WithStackOnce(err)
	}

	err = s.CreateObject(obj.(runtime.Object))
	if err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) CreateObject(obj runtime.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	err := s.client.Create(ctx, obj.(ctrlcli.Object))
	if err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) UpdateObject(obj ctrlcli.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	if err := s.client.Update(ctx, obj); err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) PatchObject(obj ctrlcli.Object, patch ctrlcli.Object) error {
	data, err := json.Marshal(patch)
	if err != nil {
		return errx.WithStackOnce(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	if err := s.client.Patch(ctx, obj, ctrlcli.RawPatch(types.StrategicMergePatchType, data)); err != nil {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) WriteStatus(obj ctrlcli.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	err := s.client.Status().Update(ctx, obj)
	if err != nil {
		s.log.Warnf("err: %+v", err)

		// may be it's k8s v1.10 and erlier (e.g. oc3.9) that doesn't support status updates
		// so try to update whole CR
		ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
		defer cancel()
		err := s.client.Update(ctx, obj)
		if err != nil {
			s.log.Warnf("err: %+v", err)
			return errors2.WithStack(rerr.Err_wait_requeue)
		}
	}

	return nil
}

func (s *Kcli) Ensure(obj metav1.Object, found ctrlcli.Object) error {
	ok, err := s.IsExists(obj, found)
	if err != nil {
		return errx.WithStackOnce(err)
	}
	if ok {
		return nil
	}

	err = s.SetRefAndCreateObject(obj)
	if k8serr.IsAlreadyExists(errors2.Unwrap(err)) {
		return nil
	}

	return nil
}

// not set ownerRef，用于公用资源
func (s *Kcli) EnsureOrphan(obj metav1.Object, found ctrlcli.Object) error {
	ok, err := s.IsExists(obj, found)
	if err != nil {
		return errx.WithStackOnce(err)
	}
	if ok {
		return nil
	}

	err = s.CreateObject(obj.(runtime.Object))
	if k8serr.IsAlreadyExists(errors2.Unwrap(err)) {
		return nil
	}

	return nil
}

func (s *Kcli) DeleteObject(obj ctrlcli.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	err := s.client.Delete(ctx, obj)
	if err != nil && !k8serr.IsNotFound(err) {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) DeleteALLByLabel(obj ctrlcli.Object, namespace string, labels map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), CtxTimeout)
	defer cancel()
	err := s.client.DeleteAllOf(ctx, obj, ctrlcli.InNamespace(namespace), ctrlcli.MatchingLabels(labels))
	if err != nil && !k8serr.IsNotFound(err) {
		return errx.WithStackOnce(err)
	}

	return nil
}

func (s *Kcli) EnsurePVC(obj *corev1.PersistentVolumeClaim) error {
	if obj == nil {
		// 不需要pvc
		return nil
	}

	found := &corev1.PersistentVolumeClaim{}
	return s.Ensure(obj, found)
}

func (s *Kcli) EnsureStatefulSet(obj *appsv1.StatefulSet) error {
	found := &appsv1.StatefulSet{}
	return s.Ensure(obj, found)
}

func (s *Kcli) EnsureService(obj *corev1.Service) error {
	found := &corev1.Service{}
	return s.Ensure(obj, found)
}
