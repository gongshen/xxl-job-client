package handler

type ExecutorBlockStrategyErr struct {
	msg string
}

func (*ExecutorBlockStrategyErr) Temporary() bool {
	return true
}

func (e *ExecutorBlockStrategyErr) Error() string {
	return e.msg
}
