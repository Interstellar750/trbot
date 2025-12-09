package flaterr

import (
	"fmt"
	"log"
	"strings"

	"github.com/rs/zerolog"
)

type Wrapper struct {
	event   *zerolog.Event
	multErr *MultErr

	isDiscard bool

	err error

	fields
}

type fields struct {
	mBool    map[string]bool
	mInt     map[string]int
	mInt64   map[string]int64
	mFloat64 map[string]float64
	mStr     map[string]string
}

// NewWrapper 传入一个 zerolog.Event 和 MultErr, 返回一个 Wrapper 等待后续链式调用
func NewWrapper(errEvent *zerolog.Event) *Wrapper {
	return &Wrapper{
		event:   errEvent,
		multErr: &MultErr{},
	}
}

// Flat 返回 multErr 中的 Flat 方法
func (ew *Wrapper) Flat() error {
	return ew.multErr.Flat()
}

// Err 记录错误等待后续链式调用
func (ew *Wrapper) Err(err error) *Wrapper {
	newEvent := *ew.event
	return &Wrapper{
		event:   newEvent.Err(err),
		multErr: ew.multErr,
		err:     err,
	}
}

// ErrIf 如果 err 为 nil，则忽略后续的链式调用
func (ew *Wrapper) ErrIf(err error) *Wrapper {
	if err == nil {
		return &Wrapper{ isDiscard: true }
	}
	newEvent := *ew.event
	return &Wrapper{
		event:   newEvent.Err(err),
		multErr: ew.multErr,
		err:     err,
	}
}

// Msg 为错误的描述，作为 err 的前缀
func (ew *Wrapper) Msg(msg string) {
	if ew.isDiscard { return }

	// ew.buildFields()
	fieldStr := ew.buildFieldsString()

	if msg != "" {
		// 没有模板用 msg 和 error-wrapping
		// ew.multErr.Addf("%s: %w", msg, ew.err)
		ew.multErr.Addf("%s%s: %w", msg, fieldStr, ew.err)

		// 按模板发日志
		ew.event.Msg(msg)
	} else {
		// 什么都没有，直接塞到 multErr 里
		ew.multErr.Add(ew.err)

		// 直接发日志
		ew.event.Send()
	}

}

func (ew *Wrapper) MsgT(tmpl Msg, cont string) {
	if ew.isDiscard { return }

	ew.buildFields()
	ew.event = ew.event.Str("content", cont)

	// 有模板用模板
	ew.multErr.Addt(tmpl, cont, ew.err)

	// 按模板发日志
	ew.event.Msg(tmpl.Str())
}

func (ew *Wrapper) buildFields() {
	for k, v := range ew.mBool    { ew.event.Bool(k, v) }
	for k, v := range ew.mInt     { ew.event.Int(k, v) }
	for k, v := range ew.mInt64   { ew.event.Int64(k, v) }
	for k, v := range ew.mFloat64 { ew.event.Float64(k, v) }
	for k, v := range ew.mStr     { ew.event.Str(k, v) }
}

func (ew *Wrapper) buildFieldsString() string {
	var sb strings.Builder
	sb.WriteString(` {"fields": {`)
	for k, v := range ew.mBool    { ew.event.Bool(k, v);    sb.WriteString(fmt.Sprintf(`"%s": %v, `, k, v)) }
	for k, v := range ew.mInt     { ew.event.Int(k, v);     sb.WriteString(fmt.Sprintf(`"%s": %v, `, k, v)) }
	for k, v := range ew.mInt64   { ew.event.Int64(k, v);   sb.WriteString(fmt.Sprintf(`"%s": %v, `, k, v)) }
	for k, v := range ew.mFloat64 { ew.event.Float64(k, v); sb.WriteString(fmt.Sprintf(`"%s": %v, `, k, v)) }
	for k, v := range ew.mStr     { ew.event.Str(k, v);     sb.WriteString(fmt.Sprintf(`"%s": "%v", `, k, v)) }
	return strings.TrimRight(sb.String(), ", ") + "}}"
}
