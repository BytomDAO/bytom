package envload

import "context"

func (o *option) Name() string {
	return o.name
}

func (o *option) Value() interface{} {
	return o.value
}

// WithLoadEnvdir specifies if Loader should load the original
// environment variables AND the contents of envdir
func WithLoadEnvdir(b bool) Option {
	return &option{
		name:  LoadEnvdirKey,
		value: b,
	}
}
func WithContext(ctx context.Context) Option {
	return &option{
		name:  ContextKey,
		value: ctx,
	}
}

func WithEnvironment(e Environment) Option {
	return &option{
		name:  EnvironmentKey,
		value: e,
	}
}
