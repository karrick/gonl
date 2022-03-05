package gonl

type ErrClose struct {
	err error
}

func (e ErrClose) Error() string {
	return "cannot close: " + e.err.Error()
}

func (e ErrClose) Unwrap() error {
	return e.err
}

type ErrRead struct {
	err error
}

func (e ErrRead) Error() string {
	return "cannot read: " + e.err.Error()
}

func (e ErrRead) Unwrap() error {
	return e.err
}

type ErrWrite struct {
	err error
}

func (e ErrWrite) Error() string {
	return "cannot write: " + e.err.Error()
}

func (e ErrWrite) Unwrap() error {
	return e.err
}
