package scd

import "errors"

var (
	FindOutOfIndex = errors.New("the specified index is out of the data range")
)
