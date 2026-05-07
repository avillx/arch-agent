package types

import "errors"

var ErrIsNotExist = errors.New("is not exist")
var ErrMistake = errors.New("worng usage")
var ErrBadToolCall = errors.New("bad tool call")
