package concurrency

import (
	"log/slog"
	"runtime/debug"
)

// SafeGo runs a function in a goroutine with panic recovery.
func SafeGo(fn func(), onPanic func(interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				slog.Error("Panic recovered", "panic", r, "stack", string(stack))
				if onPanic != nil {
					onPanic(r)
				}
			}
		}()
		fn()
	}()
}
