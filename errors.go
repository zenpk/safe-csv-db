package scd

import "errors"

var (
	FindOutOfIndex = errors.New("the specified column number is out of range")
	ValueNotFound  = errors.New("cannot find the matched value")
)
