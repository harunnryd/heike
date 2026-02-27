package adapter

import "context"

type NullAdapter struct {
	name string
}

func NewNullAdapter(name string) *NullAdapter {
	if name == "" {
		name = "null"
	}
	return &NullAdapter{name: name}
}

func (a *NullAdapter) Name() string {
	return a.name
}

func (a *NullAdapter) Send(ctx context.Context, sessionID string, content string) error {
	return nil
}

func (a *NullAdapter) Health(ctx context.Context) error {
	return nil
}
