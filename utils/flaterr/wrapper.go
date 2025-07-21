package flaterr

import (
	"fmt"

	"github.com/rs/zerolog"
)

type ErrWrapper struct {
	event        zerolog.Event
	err          error

	flatErr     *MultErr
	errForflat   error

	errContent   string
	msgTemplate  Msg

	mBool     map[string]bool
	mInt      map[string]int
	mInt64    map[string]int64
	mFloat64  map[string]float64
	mStr      map[string]string
}


func LogWithErr(zerolog *zerolog.Logger, mErr *MultErr) *ErrWrapper {
	return &ErrWrapper{
		event: *zerolog.Error(),
		flatErr:  mErr,

		mBool:    map[string]bool{},
		mInt:     map[string]int{},
		mInt64:   map[string]int64{},
		mFloat64: map[string]float64{},
		mStr:     map[string]string{},
	}
}

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

// 以错误模板发送日志和添加错误
func (ew *ErrWrapper) Tpl(msg Msg) *ErrWrapper {
	ew.msgTemplate = msg
	return ew
}

// log 信息，错误信息
func (ew *ErrWrapper) Msg(log string) *ErrWrapper {
	ew.mStr["message"] = log
	return ew
}

func (ew *ErrWrapper) Addf(format string, a ...any) *ErrWrapper {
	ew.errForflat = fmt.Errorf(format, a...)
	return ew
}

// 错误内容 错误模板

// 将字段注入 zerolog.Event 并返回一个空的函数
func (ew *ErrWrapper) Done(){
	// ew.event.Msg("")
	if ew.errForflat != nil {
		ew.flatErr.Add(ew.errForflat)
	}
	if ew.err != nil {
		ew.event.Err(ew.err)
	}
	if ew.errContent != "" {
		ew.event.Str("content", ew.errContent)
	}
	for k, v := range ew.mBool    { ew.event.Bool(k, v) }
	for k, v := range ew.mInt     { ew.event.Int(k, v) }
	for k, v := range ew.mInt64   { ew.event.Int64(k, v) }
	for k, v := range ew.mStr     { ew.event.Str(k, v) }
	for k, v := range ew.mFloat64 { ew.event.Float64(k, v) }
}
