package v1

import (
	log "github.com/win5do/go-lib/logx"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/win5do/etcd-operator/pkg/conf"
)

func whLog() *zap.SugaredLogger {
	return log.With("webhook", "Etcd")
}

func (in *Etcd) SetupWebhookWithManager(mgr ctrl.Manager) error {
	whLog().Info("setup webhook")

	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

var _ webhook.Defaulter = &Etcd{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Etcd) Default() {
	whLog().Debug("set default")

	cfg := conf.GetGlobalConfig()
	specEnv := in.Spec.Env
	defer func() {
		in.Spec.Env = specEnv
	}()

	if in.Spec.Members <= 0 {
		in.Spec.Members = 3
	}

	if in.Spec.Image == "" {
		in.Spec.Image = cfg.IMAGE
	}

	if in.Spec.ExternalHost == "" {
		in.Spec.ExternalHost = cfg.EXTERNAL_DOMAIN
	}

	if in.Spec.StorageClassName == "" {
		in.Spec.StorageClassName = cfg.STORAGE_CLASS_NAME
	}

	specEnv = MergeEnv(specEnv, cfg.InstanceEnv)
}

var _ webhook.Validator = &Etcd{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Etcd) ValidateCreate() error {
	whLog().Info("validate create", "name", in.Name)

	return in.validateCr()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Etcd) ValidateUpdate(old runtime.Object) error {
	whLog().Info("validate update", "name", in.Name)

	oldCr := old.(*Etcd)

	arrErrs := validation.ValidateImmutableField(in.Spec.Members, oldCr.Spec.Members, field.NewPath("spec").Child("nodeNumber"))

	if len(arrErrs) > 0 {
		return arrErrs[0]
	}

	return in.validateCr()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Etcd) ValidateDelete() error {
	whLog().Info("validate delete", "name", in.Name)

	return nil
}

func (in *Etcd) validateCr() error {
	var allErrs field.ErrorList
	if err := in.validateSpec(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "db.gogo.io", Kind: "Etcd"},
		in.Name, allErrs)
}

func (in *Etcd) validateSpec() *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.

	err := validateResource(in.Spec.Cpu, field.NewPath("spec").Child("cpu"))
	if err != nil {
		return err
	}

	err = validateResource(in.Spec.Memory, field.NewPath("spec").Child("memory"))
	if err != nil {
		return err
	}

	err = validateResource(in.Spec.Storage, field.NewPath("spec").Child("storage"))
	if err != nil {
		return err
	}

	return nil
}

func validateResource(val string, fldPath *field.Path) *field.Error {
	if val == "" {
		return nil
	}

	if _, err := resource.ParseQuantity(val); err != nil {
		return field.Invalid(fldPath, val, err.Error())
	}
	return nil
}

// "2Gi" -> 2048
// "1024Mi" -> 1024
func memQuantityToMiInt(s string) int64 {
	return quantityStrToInt(s) / 1000 / (1024 * 1024)
}

// "4000m" -> 4
// "4" -> 4
// "400m" -> 0.4
func cpuQuantityToInt(s string) float64 {
	return float64(quantityStrToInt(s)) / 1000
}

func quantityStrToInt(s string) int64 {
	quantity, err := resource.ParseQuantity(s)
	if err != nil {
		return -1
	}

	return quantity.MilliValue()
}
