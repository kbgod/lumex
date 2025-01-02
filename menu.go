package lumex

type IMenu interface {
	Unwrap() ReplyMarkup
}

type Menu struct {
	ReplyKeyboardMarkup
	rowIndex int
}

func WithMenuKeyboardResize(resize bool) MenuOption {
	return func(menu *Menu) {
		menu.ResizeKeyboard = resize
	}
}

type MenuOption func(*Menu)

func NewMenu(options ...MenuOption) *Menu {
	menu := &Menu{
		ReplyKeyboardMarkup: ReplyKeyboardMarkup{
			ResizeKeyboard: true,
			Keyboard:       make([][]KeyboardButton, 1),
		},
	}
	for _, option := range options {
		option(menu)
	}
	return menu
}

func (m *Menu) Unwrap() ReplyMarkup {
	return m.ReplyKeyboardMarkup
}

func (m *Menu) SetPersistent(isPersistent bool) *Menu {
	m.IsPersistent = isPersistent

	return m
}

func (m *Menu) SetResize(resize bool) *Menu {
	m.ResizeKeyboard = resize

	return m
}

func (m *Menu) SetOneTime(isOneTime bool) *Menu {
	m.OneTimeKeyboard = isOneTime

	return m
}

func (m *Menu) SetPlaceholder(text string) *Menu {
	m.InputFieldPlaceholder = text

	return m
}

func (m *Menu) SetSelective(selective bool) *Menu {
	m.Selective = selective

	return m
}

func (m *Menu) Row(buttons ...KeyboardButton) *Menu {
	if len(m.Keyboard[m.rowIndex]) == 0 {
		m.Keyboard[m.rowIndex] = buttons
	} else {
		m.Keyboard = append(m.Keyboard, buttons)
		m.rowIndex++
	}

	return m
}

func (m *Menu) TextRow(buttons ...string) *Menu {
	keyboardButtons := make([]KeyboardButton, 0, len(buttons))
	for _, button := range buttons {
		keyboardButtons = append(keyboardButtons, KeyboardButton{
			Text: button,
		})
	}
	m.Row(keyboardButtons...)

	return m
}

func (m *Menu) Fill(perLine int, buttons ...KeyboardButton) *Menu {
	for i := 0; i < len(buttons); i += perLine {
		end := i + perLine
		if end > len(buttons) {
			end = len(buttons)
		}
		m.Row(buttons[i:end]...)
	}

	return m
}

func (m *Menu) TextFill(perLine int, buttons ...string) *Menu {
	keyboardButtons := make([]KeyboardButton, 0, len(buttons))
	for _, button := range buttons {
		keyboardButtons = append(keyboardButtons, KeyboardButton{
			Text: button,
		})
	}
	m.Fill(perLine, keyboardButtons...)

	return m
}

func (m *Menu) Btn(btn KeyboardButton) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], btn)

	return m
}

func (m *Menu) TextBtn(text string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text: text,
	})

	return m
}

func (m *Menu) RequestQuizBtn(text string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text: text,
		RequestPoll: &KeyboardButtonPollType{
			Type: "quiz",
		},
	})
	return m
}

func (m *Menu) RequestPollBtn(text string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text: text,
		RequestPoll: &KeyboardButtonPollType{
			Type: "regular",
		},
	})

	return m
}

func (m *Menu) ContactBtn(text string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text:           text,
		RequestContact: true,
	})

	return m
}

func (m *Menu) LocationBtn(text string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text:            text,
		RequestLocation: true,
	})

	return m
}

func (m *Menu) WebAppBtn(text, url string) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text: text,
		WebApp: &WebAppInfo{
			Url: url,
		},
	})

	return m
}

func (m *Menu) RequestChatBtn(text string, req *KeyboardButtonRequestChat) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text:        text,
		RequestChat: req,
	})

	return m
}

func (m *Menu) RequestUserBtn(text string, req *KeyboardButtonRequestUsers) *Menu {
	m.Keyboard[m.rowIndex] = append(m.Keyboard[m.rowIndex], KeyboardButton{
		Text:         text,
		RequestUsers: req,
	})

	return m
}

type InlineMenu struct {
	InlineKeyboardMarkup
	rowIndex int
}

func NewInlineMenu() *InlineMenu {
	menu := &InlineMenu{
		InlineKeyboardMarkup: InlineKeyboardMarkup{
			InlineKeyboard: make([][]InlineKeyboardButton, 1),
		},
	}

	return menu
}

func (m *InlineMenu) Unwrap() ReplyMarkup {
	return m.InlineKeyboardMarkup
}

func (m *InlineMenu) Row(buttons ...InlineKeyboardButton) *InlineMenu {
	if len(m.InlineKeyboard[m.rowIndex]) == 0 {
		m.InlineKeyboard[m.rowIndex] = buttons
	} else {
		m.InlineKeyboard = append(m.InlineKeyboard, buttons)
		m.rowIndex++
	}

	return m
}

func (m *InlineMenu) Fill(perLine int, buttons ...InlineKeyboardButton) *InlineMenu {
	for i := 0; i < len(buttons); i += perLine {
		end := i + perLine
		if end > len(buttons) {
			end = len(buttons)
		}
		m.Row(buttons[i:end]...)
	}

	return m
}

func (m *InlineMenu) Btn(btn InlineKeyboardButton) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], btn)

	return m
}

func (m *InlineMenu) CallbackBtn(text, data string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text:         text,
		CallbackData: data,
	})

	return m
}

func (m *InlineMenu) URLBtn(text, url string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text: text,
		Url:  url,
	})

	return m
}

func (m *InlineMenu) LoginBtn(text, loginURL string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text: text,
		LoginUrl: &LoginUrl{
			Url: loginURL,
		},
	})

	return m
}

func (m *InlineMenu) SwitchInlineQueryBtn(text, query string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text:              text,
		SwitchInlineQuery: &query,
	})

	return m
}

func (m *InlineMenu) SwitchInlineCurrentChatBtn(text, query string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text:                         text,
		SwitchInlineQueryCurrentChat: &query,
	})

	return m
}

func (m *InlineMenu) SwitchInlineChosenChatBtn(
	text string, query *SwitchInlineQueryChosenChat,
) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text:                        text,
		SwitchInlineQueryChosenChat: query,
	})

	return m
}

func (m *InlineMenu) GameBtn(text string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text:         text,
		CallbackGame: &CallbackGame{},
	})

	return m
}

func (m *InlineMenu) PayBtn(text string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text: text,
		Pay:  true,
	})

	return m
}

func (m *InlineMenu) WebAppBtn(text, url string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text: text,
		WebApp: &WebAppInfo{
			Url: url,
		},
	})

	return m
}

func (m *InlineMenu) CopyBtn(text, copyText string) *InlineMenu {
	m.InlineKeyboard[m.rowIndex] = append(m.InlineKeyboard[m.rowIndex], InlineKeyboardButton{
		Text: text,
		CopyText: &CopyTextButton{
			Text: copyText,
		},
	})

	return m
}

func CallbackBtn(text, data string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: data,
	}
}

func NewForceReply() *ForceReply {
	return &ForceReply{
		ForceReply: true,
	}
}

func (v *ForceReply) Unwrap() ReplyMarkup {
	return v
}

func (v *ForceReply) SetSelective(selective bool) *ForceReply {
	v.Selective = selective
	return v
}

func (v *ForceReply) SetPlaceholder(text string) *ForceReply {
	v.InputFieldPlaceholder = text
	return v
}

func NewRemoveKeyboard() *ReplyKeyboardRemove {
	return &ReplyKeyboardRemove{
		RemoveKeyboard: true,
	}
}

func (v *ReplyKeyboardRemove) Unwrap() ReplyMarkup {
	return v
}
