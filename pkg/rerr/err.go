package rerr

import (
	"errors"
)

var (
	Err_wait_requeue  = errors.New("wait requeue")
	Err_status_not_ok = errors.New("status not ok")
)
