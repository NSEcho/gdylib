package gdylib

import "errors"

var (
	ErrFileNotSupported = errors.New("file not supported")
	ErrNotExecute       = errors.New("not executable (MH_EXECUTE)")
	ErrNotLastCommand   = errors.New("cmd LC_CODE_SIGNATURE not last")
	ErrTypeNotSupported = errors.New("unsupported load type")
	ErrNotEnoughSpace   = errors.New("not enough space for new command")
)
