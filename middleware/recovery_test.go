package middleware

import (
	"context"
	"testing"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/mocks"
	"github.com/kbgod/lumex/v2/router"
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
