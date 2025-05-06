package main

import (
	"github.com/zekrotja/ken"
	"slices"
)

type Middleware struct{}

var _ = (*ken.Middleware)(nil)

func (m *Middleware) Before(ctx *ken.Ctx) (next bool, err error) {
	next = slices.Contains(*allowedUsers, ctx.User().ID)
	if !next {
		err = ctx.RespondError("You are not allowed to use this command", "missing permissions")
	}

	return
}

func (m *Middleware) After(ctx *ken.Ctx, cmdError error) (err error) {
	if cmdError != nil {
		err = ctx.RespondError("`"+cmdError.Error()+"`", "There was an error")
	}
	return
}
