package worklog

import "fmt"

// ErrorKind lets the CLI map library errors to stable exit codes.
type ErrorKind string

const (
	KindConfig  ErrorKind = "config"
	KindUsage   ErrorKind = "usage"
	KindNetwork ErrorKind = "network"
	KindRemote  ErrorKind = "remote"
	KindRefused ErrorKind = "refused"
)

type Error struct {
	Kind ErrorKind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	if e.Msg == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Msg, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func configError(format string, args ...any) error {
	return &Error{Kind: KindConfig, Msg: fmt.Sprintf(format, args...)}
}

func usageError(format string, args ...any) error {
	return &Error{Kind: KindUsage, Msg: fmt.Sprintf(format, args...)}
}

func remoteError(format string, args ...any) error {
	return &Error{Kind: KindRemote, Msg: fmt.Sprintf(format, args...)}
}

func networkError(err error) error {
	return &Error{Kind: KindNetwork, Msg: "网络请求失败", Err: err}
}

func refusedError(format string, args ...any) error {
	return &Error{Kind: KindRefused, Msg: fmt.Sprintf(format, args...)}
}

// The exported constructors are used by the CLI command layer.
func UsageError(format string, args ...any) error   { return usageError(format, args...) }
func ConfigError(format string, args ...any) error  { return configError(format, args...) }
func RemoteError(format string, args ...any) error  { return remoteError(format, args...) }
func RefusedError(format string, args ...any) error { return refusedError(format, args...) }
