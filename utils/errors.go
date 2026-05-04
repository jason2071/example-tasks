package utils

import (
	"errors"
)

var ErrTaskNotFound200 = errors.New("resource not found")
var ErrTaskAlreadyExists200 = errors.New("task with the same title already exists")
