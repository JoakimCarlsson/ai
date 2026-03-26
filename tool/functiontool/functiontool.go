// Package functiontool wraps plain Go functions as tool.BaseTool implementations.
package functiontool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/joakimcarlsson/ai/tool"
)

var (
	ctxType      = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType    = reflect.TypeOf((*error)(nil)).Elem()
	responseType = reflect.TypeOf(tool.Response{})
)

type returnStyle int

const (
	returnString returnStyle = iota
	returnResponse
	returnJSON
)

// Option configures a function tool created by New.
type Option func(*funcTool)

// WithConfirmation marks the tool as requiring human approval before execution.
func WithConfirmation() Option {
	return func(ft *funcTool) {
		ft.info.RequireConfirmation = true
	}
}

type funcTool struct {
	info        tool.Info
	fn          reflect.Value
	hasCtx      bool
	paramType   reflect.Type
	returnStyle returnStyle
}

func (ft *funcTool) Info() tool.Info { return ft.info }

func (ft *funcTool) Run(
	ctx context.Context,
	call tool.Call,
) (tool.Response, error) {
	var args []reflect.Value

	if ft.hasCtx {
		args = append(args, reflect.ValueOf(ctx))
	}

	if ft.paramType != nil {
		paramPtr := reflect.New(ft.paramType)
		if call.Input != "" {
			if err := json.Unmarshal([]byte(call.Input), paramPtr.Interface()); err != nil {
				return tool.NewTextErrorResponse(
					"invalid input: " + err.Error(),
				), nil
			}
		}
		args = append(args, paramPtr.Elem())
	}

	var (
		results []reflect.Value
		fnErr   error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fnErr = fmt.Errorf("tool panicked: %v", r)
			}
		}()
		results = ft.fn.Call(args)
	}()

	if fnErr != nil {
		return tool.NewTextErrorResponse(fnErr.Error()), nil
	}

	if errVal := results[1]; !errVal.IsNil() {
		fnErr = errVal.Interface().(error)
	}

	if fnErr != nil {
		if errors.Is(fnErr, tool.ErrConfirmationRejected) {
			return tool.Response{}, fnErr
		}
		return tool.NewTextErrorResponse(fnErr.Error()), nil
	}

	switch ft.returnStyle {
	case returnString:
		return tool.NewTextResponse(results[0].String()), nil
	case returnResponse:
		return results[0].Interface().(tool.Response), nil
	case returnJSON:
		return tool.NewJSONResponse(results[0].Interface()), nil
	default:
		return tool.NewTextErrorResponse("unknown return style"), nil
	}
}

// New wraps fn as a tool.BaseTool. The function's schema is inferred from
// its parameter struct type. Panics if fn does not match a supported signature.
func New(name, description string, fn any, opts ...Option) tool.BaseTool {
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		panic("functiontool.New: fn must be a function")
	}

	numIn := fnType.NumIn()
	if numIn > 2 {
		panic(
			"functiontool.New: fn must have at most 2 parameters (context.Context, ParamsStruct)",
		)
	}

	var (
		hasCtx    bool
		paramType reflect.Type
	)

	idx := 0
	if numIn > idx && fnType.In(idx).Implements(ctxType) {
		hasCtx = true
		idx++
	}

	if numIn > idx {
		pt := fnType.In(idx)
		if pt.Kind() == reflect.Ptr {
			pt = pt.Elem()
		}
		if pt.Kind() != reflect.Struct {
			panic("functiontool.New: parameter type must be a struct")
		}
		paramType = pt
		idx++
	}

	if idx != numIn {
		panic(
			"functiontool.New: unexpected parameter types; expected ([context.Context], [ParamsStruct])",
		)
	}

	if fnType.NumOut() != 2 {
		panic(
			"functiontool.New: fn must return exactly 2 values (result, error)",
		)
	}
	if !fnType.Out(1).Implements(errorType) {
		panic("functiontool.New: fn's second return value must implement error")
	}

	var rs returnStyle
	switch fnType.Out(0) {
	case reflect.TypeOf(""):
		rs = returnString
	case responseType:
		rs = returnResponse
	default:
		rs = returnJSON
	}

	var info tool.Info
	if paramType != nil {
		info = tool.NewInfo(
			name,
			description,
			reflect.New(paramType).Elem().Interface(),
		)
	} else {
		info = tool.Info{
			Name:        name,
			Description: description,
		}
	}

	ft := &funcTool{
		info:        info,
		fn:          reflect.ValueOf(fn),
		hasCtx:      hasCtx,
		paramType:   paramType,
		returnStyle: rs,
	}

	for _, opt := range opts {
		opt(ft)
	}

	return ft
}
