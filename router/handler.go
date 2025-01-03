package router

type Handler func(*Context) error
type ErrorHandler func(*Context, error)
