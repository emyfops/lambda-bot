package main

import (
	"github.com/zekrotja/ken"
)

type Middleware struct{}

var _ ken.MiddlewareAfter = (*Middleware)(nil)

func (m *Middleware) After(ctx *ken.Ctx, cmdError error) (err error) {
	if cmdError != nil {
		err = ctx.RespondError("`"+cmdError.Error()+"`", "There was an error")
	}
	return
}
