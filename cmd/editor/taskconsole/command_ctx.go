package taskconsole

import "context"

type CancelFunc func(err error)

type commandContext struct {
	context.Context
	lastErr error
}

func (c *commandContext) Err() error {
	return c.lastErr
}

func newCommandContext() (context.Context, CancelFunc) {
	innerCtx, cancel := context.WithCancel(context.Background())
	ctx := &commandContext{
		Context: innerCtx,
	}
	return ctx, func(err error) {
		ctx.lastErr = err
		cancel()
	}
}
