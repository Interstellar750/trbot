package flaterr

// return "content", str
func Cont(str string) (string, string) {
	return "content", str

}

func (ew *ErrWrapper) Cont(content string) *ErrWrapper {
	if content != "" {
		ew.errContent = content
		ew.event.Str("content", ew.errContent)
	}
	return ew
}

func (ew *ErrWrapper) Err(err error) *ErrWrapper {
	if err != nil {
		ew.err = err
		ew.event.Err(err)
	}
	return ew
}

func (ew *ErrWrapper) Bool(key string, val bool) *ErrWrapper {
	if key != "" {
		ew.event.Bool(key, val)
		ew.mBool[key] = val
	}
	return ew
}

func (ew *ErrWrapper) Int(key string, val int) *ErrWrapper {
	if key != "" {
		ew.event.Int(key, val)
		ew.mInt[key] = val
	}
	return ew
}

func (ew *ErrWrapper) Int64(key string, val int64) *ErrWrapper {
	if key != "" {
		ew.event.Int64(key, val)
		ew.mInt64[key] = val
	}
	return ew
}

func (ew *ErrWrapper) Str(key string, val string) *ErrWrapper {
	if key != "" {
		ew.event.Str(key, val)
		ew.mStr[key] = val
	}
	return ew
}
