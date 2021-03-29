/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
	"github.com/win5do/etcd-operator/pkg/conf"
	"github.com/win5do/etcd-operator/pkg/controller"
	"github.com/win5do/etcd-operator/pkg/rerr"
)

// EtcdReconciler reconciles a Etcd object
type EtcdReconciler struct {
	client.Client
	Log    *zap.SugaredLogger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=db.gogo.io,resources=etcds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=db.gogo.io,resources=etcds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=db.gogo.io,resources=etcds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *EtcdReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := r.Log.With("etcd", req.NamespacedName)

	// your logic here
	cr := &dbv1.Etcd{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	return r.reconcile(rlog, cr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbv1.Etcd{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *EtcdReconciler) reconcile(rlog *zap.SugaredLogger, cr *dbv1.Etcd) (reconcile.Result, error) {
	herr := rerr.NewHandler(rlog)

	ct := controller.Inject(r.Client, r.Scheme, cr, rlog, conf.GetGlobalConfig())

	// ---> delete & clean
	{
		if cr.GetDeletionTimestamp() != nil {
			err := ct.Finalize()
			return herr.HandleErr(err)
		}

		err := ct.AddFinalizer()
		if err != nil {
			return herr.HandleErr(err)
		}
	}

	// ---> sync svc
	{
		err := ct.SyncSvc()
		if err != nil {
			return herr.HandleErr(err)
		}
	}

	// ---> headless svc, work fine with p8s
	{
		err := ct.Kcli.EnsureService(ct.Builder.HeadlessService(
			cr.Name,
			controller.MemberLabel(cr.ObjectMeta, controller.SelectAll),
			controller.MemberLabel(cr.ObjectMeta, controller.SelectAll),
		))
		if err != nil {
			return herr.HandleErr(err)
		}
	}

	// ---> sync sts
	{
		newSts := ct.Builder.StatefulSet(controller.MemberLabel(cr.ObjectMeta, controller.SelectAll))
		oldSts := &appsv1.StatefulSet{}
		err := ct.Kcli.Ensure(newSts, oldSts)
		if err != nil {
			return herr.HandleErr(err)
		}

		if oldSts.UID != "" {
			if oldSts.Annotations[controller.SpecHash] != newSts.Annotations[controller.SpecHash] {
				err := ct.Kcli.PatchObject(oldSts, &appsv1.StatefulSet{
					ObjectMeta: newSts.ObjectMeta,
					Spec:       newSts.Spec,
				})
				if err != nil {
					return herr.HandleErr(err)
				}
			}
		}
	}

	// ---> set status
	{
		nodePorts, err := ct.ListSvcNodePort(controller.PortClientName, cr.Namespace, controller.ExportSvcLabel(cr.ObjectMeta, controller.SelectAll))
		if err != nil {
			return herr.HandleErr(err)
		}
		rlog.Debugf("nodePorts: %v", nodePorts)

		err = ct.StatusManager.HandleStatus(cr, nodePorts)
		if err != nil {
			return herr.HandleErr(err)
		}
	}

	return herr.HandleErr(nil)
}
