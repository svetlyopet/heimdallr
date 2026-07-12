package requestlimits

import "context"

const DefaultMaxDecodedOutputBytes int64 = 4 << 20

type contextKey struct{}

type Values struct {
	MaxDecodedOutputBytes int64
}

func WithContext(ctx context.Context, values Values) context.Context {
	if values.MaxDecodedOutputBytes <= 0 {
		values.MaxDecodedOutputBytes = DefaultMaxDecodedOutputBytes
	}

	return context.WithValue(ctx, contextKey{}, values)
}

func MaxDecodedOutputBytes(ctx context.Context) int64 {
	if ctx != nil {
		if values, ok := ctx.Value(contextKey{}).(Values); ok && values.MaxDecodedOutputBytes > 0 {
			return values.MaxDecodedOutputBytes
		}
	}

	return DefaultMaxDecodedOutputBytes
}
