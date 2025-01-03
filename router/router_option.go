package router

import "github.com/kbgod/lumex/log"

type Option func(*Router)

// WithErrorHandler
//
// is an option for the router that sets the error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(r *Router) {
		r.errorHandler = handler
	}
}

// WithLogger
//
// is an option for the router that sets the logger.
// If not set, the router will use an empty logger.
// The logger is used to log important router warnings and errors.
func WithLogger(logger log.Logger) Option {
	return func(r *Router) {
		r.log = logger
	}
}
