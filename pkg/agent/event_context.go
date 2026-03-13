package agent

import "context"

type eventSinkContextKey struct{}
type runDebugContextKey struct{}

func withEventSink(ctx context.Context, sink func(*Event)) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, eventSinkContextKey{}, sink)
}

func eventSinkFromContext(ctx context.Context) func(*Event) {
	if ctx == nil {
		return nil
	}
	sink, _ := ctx.Value(eventSinkContextKey{}).(func(*Event))
	return sink
}

func withRunDebug(ctx context.Context, debug bool) context.Context {
	return context.WithValue(ctx, runDebugContextKey{}, debug)
}

func runDebugFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	debug, _ := ctx.Value(runDebugContextKey{}).(bool)
	return debug
}
