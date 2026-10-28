package trip

import "errors"

var (
	ErrNotFound   = errors.New("trip not found")
	ErrDriverBusy = errors.New("driver already has an active trip")
	ErrCompleted  = errors.New("trip is already completed")
)
