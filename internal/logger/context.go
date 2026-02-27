package logger

import "context"

type contextKey string

const TraceIDKey contextKey = "trace_id"
const SessionIDKey contextKey = "session_id"

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, TraceIDKey, id)
}

func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SessionIDKey, id)
}

func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}
