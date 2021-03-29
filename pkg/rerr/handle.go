package rerr

import (
	"time"

	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Handler struct {
	log *zap.SugaredLogger
}

func NewHandler(log *zap.SugaredLogger) *Handler {
	return &Handler{
		log: log,
	}
}

func (s *Handler) HandleErr(err error) (ctrl.Result, error) {
	rlog := s.log

	if err == nil {
		return ctrl.Result{}, nil
	}

	r := ctrl.Result{
		Requeue:      true,
		RequeueAfter: 3 * time.Second,
	}

	if errors2.Is(err, Err_wait_requeue) {
		rlog.Debugf("requeue: %+v", err.Error())
		return r, nil
	} else {
		rlog.Errorf("err: %+v", err)
		return r, err
	}
}
