package lumex

import (
	"testing"
)

func TestNewMenu(t *testing.T) {
	menu := NewMenu()
	if menu == nil {
		t.Error("NewMenu failed")
	}
	if menu.Keyboard == nil {
		t.Error("NewMenu failed")
	}

	menu = NewMenu(WithMenuKeyboardResize(false))
	if menu.ResizeKeyboard {
		t.Error("NewMenu failed")
	}
}

func TestMenu_Row(t *testing.T) {
	menu := NewMenu()
	menu.Row(KeyboardButton{})
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.Row failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.Row failed")
	}

	menu.Row(KeyboardButton{}, KeyboardButton{})
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.Row failed")
	}
	if len(menu.Keyboard[1]) != 2 {
		t.Error("Menu.Row failed")
	}
}

func TestMenu_TextRow(t *testing.T) {
	menu := NewMenu()
	menu.TextRow("test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.TextRow failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.TextRow failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.TextRow failed")
	}

	menu.TextRow("test", "test")
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.TextRow failed")
	}
	if len(menu.Keyboard[1]) != 2 {
		t.Error("Menu.TextRow failed")
	}
	if menu.Keyboard[1][0].Text != "test" {
		t.Error("Menu.TextRow failed")
	}
	if menu.Keyboard[1][1].Text != "test" {
		t.Error("Menu.TextRow failed")
	}
}

func TestMenu_Fill(t *testing.T) {
	menu := NewMenu()
	menu.Fill(1, KeyboardButton{})
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.Fill failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.Fill failed")
	}

	menu = NewMenu()
	menu.Fill(1, KeyboardButton{}, KeyboardButton{})
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.Fill failed")
	}
	if len(menu.Keyboard[0]) != 1 || len(menu.Keyboard[1]) != 1 {
		t.Error("Menu.Fill failed")
	}

	menu = NewMenu()
	menu.Fill(2, KeyboardButton{}, KeyboardButton{}, KeyboardButton{})
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.Fill failed")
	}
	if len(menu.Keyboard[0]) != 2 || len(menu.Keyboard[1]) != 1 {
		t.Error("Menu.Fill failed")
	}
}

func TestMenu_TextFill(t *testing.T) {
	menu := NewMenu()
	menu.TextFill(1, "test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.TextFill failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.TextFill failed")
	}

	menu = NewMenu()
	menu.TextFill(1, "test", "test")
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.TextFill failed")
	}
	if len(menu.Keyboard[0]) != 1 || len(menu.Keyboard[1]) != 1 {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[1][0].Text != "test" {
		t.Error("Menu.TextFill failed")
	}

	menu = NewMenu()
	menu.TextFill(2, "test", "test", "test")
	if len(menu.Keyboard) != 2 {
		t.Error("Menu.TextFill failed")
	}
	if len(menu.Keyboard[0]) != 2 || len(menu.Keyboard[1]) != 1 {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[0][1].Text != "test" {
		t.Error("Menu.TextFill failed")
	}
	if menu.Keyboard[1][0].Text != "test" {
		t.Error("Menu.TextFill failed")
	}
}

func TestMenu_Btn(t *testing.T) {
	menu := NewMenu()
	menu.Btn(KeyboardButton{})
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.Btn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.Btn failed")
	}
}

func TestMenu_TextBtn(t *testing.T) {
	menu := NewMenu()
	menu.TextBtn("test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.TextBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.TextBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.TextBtn failed")
	}
}

func TestMenu_ContactBtn(t *testing.T) {
	menu := NewMenu()
	menu.ContactBtn("test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.ContactBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.ContactBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.ContactBtn failed")
	}
	if menu.Keyboard[0][0].RequestContact != true {
		t.Error("Menu.ContactBtn failed")
	}
}

func TestMenu_LocationBtn(t *testing.T) {
	menu := NewMenu()
	menu.LocationBtn("test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.LocationBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.LocationBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.LocationBtn failed")
	}
	if menu.Keyboard[0][0].RequestLocation != true {
		t.Error("Menu.LocationBtn failed")
	}
}

func TestMenu_WebAppBtn(t *testing.T) {
	menu := NewMenu()
	menu.WebAppBtn("test", "test")
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.WebAppBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.WebAppBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.WebAppBtn failed")
	}
	if menu.Keyboard[0][0].WebApp.Url != "test" {
		t.Error("Menu.WebAppBtn failed")
	}
}

func TestMenu_RequestChatBtn(t *testing.T) {
	menu := NewMenu()
	menu.RequestChatBtn("test", &KeyboardButtonRequestChat{})
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.RequestChatBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.RequestChatBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.RequestChatBtn failed")
	}
	if menu.Keyboard[0][0].RequestChat == nil {
		t.Error("Menu.RequestChatBtn failed")
	}
}

func TestMenu_RequestUserBtn(t *testing.T) {
	menu := NewMenu()
	menu.RequestUserBtn("test", &KeyboardButtonRequestUsers{})
	if len(menu.Keyboard) != 1 {
		t.Error("Menu.RequestUserBtn failed")
	}
	if len(menu.Keyboard[0]) != 1 {
		t.Error("Menu.RequestUserBtn failed")
	}
	if menu.Keyboard[0][0].Text != "test" {
		t.Error("Menu.RequestUserBtn failed")
	}
	if menu.Keyboard[0][0].RequestUsers == nil {
		t.Error("Menu.RequestUserBtn failed")
	}
}

func TestNewInlineMenu(t *testing.T) {
	menu := NewInlineMenu()
	if menu == nil {
		t.Error("NewInlineMenu failed")
	}
	if menu.InlineKeyboard == nil {
		t.Error("NewInlineMenu failed")
	}
}

func TestInlineMenu_Row(t *testing.T) {
	menu := NewInlineMenu()
	menu.Row(InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.Row failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.Row failed")
	}

	menu.Row(InlineKeyboardButton{}, InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 2 {
		t.Error("InlineMenu.Row failed")
	}
	if len(menu.InlineKeyboard[1]) != 2 {
		t.Error("InlineMenu.Row failed")
	}
}

func TestInlineMenu_Fill(t *testing.T) {
	menu := NewInlineMenu()
	menu.Fill(1, InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.Fill failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.Fill failed")
	}

	menu = NewInlineMenu()
	menu.Fill(1, InlineKeyboardButton{}, InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 2 {
		t.Error("InlineMenu.Fill failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 || len(menu.InlineKeyboard[1]) != 1 {
		t.Error("InlineMenu.Fill failed")
	}

	menu = NewInlineMenu()
	menu.Fill(2, InlineKeyboardButton{}, InlineKeyboardButton{}, InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 2 {
		t.Error("InlineMenu.Fill failed")
	}
	if len(menu.InlineKeyboard[0]) != 2 || len(menu.InlineKeyboard[1]) != 1 {
		t.Error("InlineMenu.Fill failed")
	}
}

func TestInlineMenu_Btn(t *testing.T) {
	menu := NewInlineMenu()
	menu.Btn(InlineKeyboardButton{})
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.Btn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.Btn failed")
	}
}

func TestInlineMenu_CallbackBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.CallbackBtn("test", "test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.CallbackBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.CallbackBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.CallbackBtn failed")
	}
	if menu.InlineKeyboard[0][0].CallbackData != "test" {
		t.Error("InlineMenu.CallbackBtn failed")
	}
}

func TestInlineMenu_URLBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.URLBtn("test", "test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.URLBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.URLBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.URLBtn failed")
	}
	if menu.InlineKeyboard[0][0].Url != "test" {
		t.Error("InlineMenu.URLBtn failed")
	}
}

func TestInlineMenu_LoginBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.LoginBtn("test", "test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.LoginBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.LoginBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.LoginBtn failed")
	}
	if menu.InlineKeyboard[0][0].LoginUrl.Url != "test" {
		t.Error("InlineMenu.LoginBtn failed")
	}
}

func TestInlineMenu_SwitchInlineBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.SwitchInlineQueryBtn("test", "test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.SwitchInlineBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.SwitchInlineBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.SwitchInlineBtn failed")
	}
	if *menu.InlineKeyboard[0][0].SwitchInlineQuery != "test" {
		t.Error("InlineMenu.SwitchInlineBtn failed")
	}
}

func TestInlineMenu_SwitchInlineCurrentBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.SwitchInlineCurrentChatBtn("test", "test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.SwitchInlineCurrentBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.SwitchInlineCurrentBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.SwitchInlineCurrentBtn failed")
	}
	if *menu.InlineKeyboard[0][0].SwitchInlineQueryCurrentChat != "test" {
		t.Error("InlineMenu.SwitchInlineCurrentBtn failed")
	}
}

func TestInlineMenu_SwitchInlineChosenChatBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.SwitchInlineChosenChatBtn("test", &SwitchInlineQueryChosenChat{})
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.SwitchInlineChosenChatBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.SwitchInlineChosenChatBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.SwitchInlineChosenChatBtn failed")
	}
	if menu.InlineKeyboard[0][0].SwitchInlineQueryCurrentChat != nil {
		t.Error("InlineMenu.SwitchInlineChosenChatBtn failed")
	}
}

func TestInlineMenu_GameBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.GameBtn("test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.GameBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.GameBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.CallbackGameBtn failed")
	}
	if menu.InlineKeyboard[0][0].CallbackGame == nil {
		t.Error("InlineMenu.GameBtn failed")
	}
}

func TestInlineMenu_PayBtn(t *testing.T) {
	menu := NewInlineMenu()
	menu.PayBtn("test")
	if len(menu.InlineKeyboard) != 1 {
		t.Error("InlineMenu.PayBtn failed")
	}
	if len(menu.InlineKeyboard[0]) != 1 {
		t.Error("InlineMenu.PayBtn failed")
	}
	if menu.InlineKeyboard[0][0].Text != "test" {
		t.Error("InlineMenu.PayBtn failed")
	}
	if !menu.InlineKeyboard[0][0].Pay {
		t.Error("InlineMenu.PayBtn failed")
	}
}

func TestCallbackBtn(t *testing.T) {
	btn := CallbackBtn("test", "test")
	if btn.Text != "test" {
		t.Error("CallbackBtn failed")
	}
	if btn.CallbackData != "test" {
		t.Error("CallbackBtn failed")
	}
}
