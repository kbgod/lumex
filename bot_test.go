package lumex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBot_NewBot(t *testing.T) {
	t.Run("without token check", func(t *testing.T) {
		bot, err := NewBot("123:abc", &BotOpts{
			DisableTokenCheck: true,
		})

		assert.NoError(t, err, "NewBot() failed")
		assert.Equal(t, int64(123), bot.Id, "unexpected bot ID")
		assert.Equal(t, true, bot.IsBot, "unexpected bot IsBot")
		assert.Equal(t, "<missing>", bot.FirstName, "unexpected bot FirstName")
		assert.Equal(t, "<missing>", bot.Username, "unexpected bot Username")
	})

	t.Run("without token check invalid token format", func(t *testing.T) {
		_, err := NewBot("invalid", &BotOpts{
			DisableTokenCheck: true,
		})

		assert.Error(t, err, "NewBot() should fail")
	})

	t.Run("without token check invalid bot id format", func(t *testing.T) {
		_, err := NewBot("abc:abc", &BotOpts{
			DisableTokenCheck: true,
		})

		assert.Error(t, err, "NewBot() should fail")
	})

	t.Run("with token check invalid token", func(t *testing.T) {
		_, err := NewBot("123:abc", &BotOpts{
			DisableTokenCheck: false,
		})

		assert.Error(t, err, "NewBot() should fail")
	})
}
