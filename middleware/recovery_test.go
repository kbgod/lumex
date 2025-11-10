package middleware

import (
	"context"
	"testing"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/mocks"
	"github.com/kbgod/lumex/router"
	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	r := router.New(nil)
	log := mocks.NewLogger(t)
	log.On("Error", nil, "fatal error", map[string]interface{}{
		"panic": "test",
	}).Once()

	r.Use(RecoveryMiddleware(log))

	r.OnUpdate(func(ctx *router.Context) error {
		panic("test")
	})

	err := r.HandleUpdate(context.Background(), &lumex.Update{})

	assert.NoError(t, err)

}
