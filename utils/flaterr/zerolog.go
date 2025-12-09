package flaterr


func (ew *Wrapper) Bool(key string, val bool) *Wrapper {
	if key != "" {
		ew.event.Bool(key, val)
		if ew.mBool == nil { ew.mBool = map[string]bool{} }
		ew.mBool[key] = val
	}
	return ew
}

func (ew *Wrapper) Int(key string, val int) *Wrapper {
	if key != "" {
		ew.event.Int(key, val)
		if ew.mInt == nil { ew.mInt = map[string]int{} }
		ew.mInt[key] = val
	}
	return ew
}

func (ew *Wrapper) Int64(key string, val int64) *Wrapper {
	if key != "" {
		ew.event.Int64(key, val)
		if ew.mInt64 == nil { ew.mInt64 = map[string]int64{} }
		ew.mInt64[key] = val
	}
	return ew
}

func (ew *Wrapper) Float64(key string, val float64) *Wrapper {
	if key != "" {
		ew.event.Float64(key, val)
		if ew.mFloat64 == nil { ew.mFloat64 = map[string]float64{} }
		ew.mFloat64[key] = val
	}
	return ew
}

func (ew *Wrapper) Str(key string, val string) *Wrapper {
	if key != "" {
		ew.event.Str(key, val)
		if ew.mStr == nil { ew.mStr = map[string]string{} }
		ew.mStr[key] = val
	}
	return ew
}
