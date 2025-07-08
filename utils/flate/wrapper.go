package flate

import (
	"github.com/rs/zerolog"
)

// 传入 MultErr, 并返回一个 ErrWrapper 等待后续链式调用
func Wrapper(handlerErr *MultErr) *ErrWrapper {
	return &ErrWrapper{
		flatErr:  handlerErr,

		mBool:    map[string]bool{},
		mInt:     map[string]int{},
		mInt64:   map[string]int64{},
		mFloat64: map[string]float64{},
		mStr:     map[string]string{},
	}
}

type ErrWrapper struct {
	flatErr     *MultErr
	err          error
	errTemplate  string
	errContent   string

	mBool     map[string]bool
	mInt      map[string]int
	mInt64    map[string]int64
	mFloat64  map[string]float64
	mStr      map[string]string
}



// 错误内容 错误模板
func (ew *ErrWrapper) ErrContent(content string, msg Msg) *ErrWrapper {
	if content != "" {
		ew.errContent = content
	}
	ew.mStr["message"] = msg.Str()
	ew.errTemplate = msg.Template()
	// ew.flatErr.Addf(errMsg, content) 移到 Done() 中
	return ew
}


func (ew *ErrWrapper) Err(err error) *ErrWrapper {
	ew.err = err
	return ew
}

func (ew *ErrWrapper) Str(key string, val string) *ErrWrapper {
	ew.mStr[key] = val
	return ew
}

// 将字段注入 zerolog.Event 并返回一个空的函数
func (ew *ErrWrapper) Done() func(*zerolog.Event) {
	if ew.errTemplate != "" && ew.errContent != "" {
		ew.flatErr.Addf(ew.errTemplate, ew.errContent)
	}
	return func(event *zerolog.Event) {
		if ew.err != nil {
			event.Err(ew.err)
		}
		if ew.errContent != "" {
			event.Str("content", ew.errContent)
		}
		for k, v := range ew.mBool    { event.Bool(k, v) }
		for k, v := range ew.mInt     { event.Int(k, v) }
		for k, v := range ew.mInt64   { event.Int64(k, v) }
		for k, v := range ew.mStr     { event.Str(k, v) }
		for k, v := range ew.mFloat64 { event.Float64(k, v) }
	}
}

func (ew *ErrWrapper) DoneAndSend() func(*zerolog.Event) {
	if ew.errTemplate != "" && ew.errContent != "" {
		ew.flatErr.Addf(ew.errTemplate, ew.errContent)
	}
	return func(event *zerolog.Event) {
		if ew.err != nil {
			event.Err(ew.err)
		}
		if ew.errContent != "" {
			event.Str("content", ew.errContent)
		}
		for k, v := range ew.mBool    { event.Bool(k, v) }
		for k, v := range ew.mInt     { event.Int(k, v) }
		for k, v := range ew.mInt64   { event.Int64(k, v) }
		for k, v := range ew.mStr     { event.Str(k, v) }
		for k, v := range ew.mFloat64 { event.Float64(k, v) }

		if ew.mStr["message"] != "" {
			event.Msg(ew.mStr["message"])
		}
	}
}
