package stream

type EventSinkerInterceptor func(EventSinker) EventSinker

func WithEventSinkerInterceptor(s EventSinker, other ...EventSinkerInterceptor) EventSinker {
	for i := len(other) - 1; i >= 0; i-- {
		interceptor := other[i]
		s = interceptor(s)
	}
	return s
}

type CommandSinkerInterceptor func(CommandSinker) CommandSinker

func WithCommandSinkerInterceptor(s CommandSinker, other ...CommandSinkerInterceptor) CommandSinker {
	for i := len(other) - 1; i >= 0; i-- {
		interceptor := other[i]
		s = interceptor(s)
	}
	return s
}
