package lumex

import (
	"unicode/utf16"
	"unicode/utf8"
)

// ParsedMessageEntity is a MessageEntity resolved against the string it
// annotates: Offset and Length are UTF-8 byte offsets into that string (not the
// UTF-16 code units Telegram sends), so text[Offset:Offset+Length] == Text.
// For a "url" entity URL is set to Text.
type ParsedMessageEntity struct {
	MessageEntity
	Text string `json:"text"`
}

// ParseEntity resolves a single entity of text. A malformed entity never panics:
// its range is clamped to text, and a boundary inside a surrogate pair moves past
// that character.
func ParseEntity(text string, entity MessageEntity) ParsedMessageEntity {
	return parseEntity(text, entity, utf16ByteOffsets(text))
}

// ParseEntities resolves every entity of text, see ParseEntity.
func ParseEntities(text string, entities []MessageEntity) []ParsedMessageEntity {
	return ParseEntityTypes(text, entities, nil)
}

// ParseEntityTypes resolves the entities of text whose type is in accepted; a nil
// accepted keeps all of them.
func ParseEntityTypes(text string, entities []MessageEntity, accepted map[MessageEntityType]struct{}) []ParsedMessageEntity {
	if len(entities) == 0 {
		return nil
	}
	var offsets []int
	var out []ParsedMessageEntity
	for _, e := range entities {
		if accepted != nil {
			if _, ok := accepted[e.Type]; !ok {
				continue
			}
		}
		if offsets == nil {
			offsets = utf16ByteOffsets(text)
		}
		out = append(out, parseEntity(text, e, offsets))
	}
	return out
}

// utf16ByteOffsets maps every UTF-16 offset of s (0..len in code units) to a byte
// offset of s. The second unit of a surrogate pair maps past the whole rune.
func utf16ByteOffsets(s string) []int {
	offsets := make([]int, 0, len(s)+1)
	for i, r := range s {
		offsets = append(offsets, i)
		if utf16.RuneLen(r) == 2 {
			offsets = append(offsets, i+utf8.RuneLen(r))
		}
	}
	return append(offsets, len(s))
}

func parseEntity(text string, entity MessageEntity, offsets []int) ParsedMessageEntity {
	total := int64(len(offsets) - 1)
	start := min(max(entity.Offset, 0), total)
	end := start + min(max(entity.Length, 0), total-start)

	from, to := offsets[start], offsets[end]
	entity.Offset, entity.Length = int64(from), int64(to-from)
	sub := text[from:to]
	if entity.Type == MessageEntityTypeURL {
		entity.URL = sub
	}
	return ParsedMessageEntity{MessageEntity: entity, Text: sub}
}

// ParseEntity resolves entity against m.Text.
func (m *Message) ParseEntity(entity MessageEntity) ParsedMessageEntity {
	if m == nil {
		return ParseEntity("", entity)
	}
	return ParseEntity(m.Text, entity)
}

// ParseCaptionEntity resolves entity against m.Caption.
func (m *Message) ParseCaptionEntity(entity MessageEntity) ParsedMessageEntity {
	if m == nil {
		return ParseEntity("", entity)
	}
	return ParseEntity(m.Caption, entity)
}

// ParseEntities resolves m.Entities against m.Text.
func (m *Message) ParseEntities() []ParsedMessageEntity {
	return m.ParseEntityTypes(nil)
}

// ParseEntityTypes resolves the m.Entities whose type is in accepted (nil = all).
func (m *Message) ParseEntityTypes(accepted map[MessageEntityType]struct{}) []ParsedMessageEntity {
	if m == nil {
		return nil
	}
	return ParseEntityTypes(m.Text, m.Entities, accepted)
}

// ParseCaptionEntities resolves m.CaptionEntities against m.Caption.
func (m *Message) ParseCaptionEntities() []ParsedMessageEntity {
	return m.ParseCaptionEntityTypes(nil)
}

// ParseCaptionEntityTypes resolves the m.CaptionEntities whose type is in
// accepted (nil = all).
func (m *Message) ParseCaptionEntityTypes(accepted map[MessageEntityType]struct{}) []ParsedMessageEntity {
	if m == nil {
		return nil
	}
	return ParseEntityTypes(m.Caption, m.CaptionEntities, accepted)
}

// ParseAnyEntities resolves the entities of every text this message carries, see
// ParseAnyEntityTypes.
func (m *Message) ParseAnyEntities() []ParsedMessageEntity {
	return m.ParseAnyEntityTypes(nil)
}

// ParseAnyEntityTypes resolves, in this order, the entities whose type is in
// accepted (nil = all) of every text that belongs to the message itself: Text,
// Caption, Checklist title and tasks, Game text, the Gift / UniqueGift /
// GiftUpgradeSent text, Poll question, options, explanation and description,
// the PollOptionAdded / PollOptionDeleted option text and the ChecklistTasksAdded
// tasks. Quote is skipped — it is part of the replied-to message. Each result's
// offsets are relative to the field it came from.
func (m *Message) ParseAnyEntityTypes(accepted map[MessageEntityType]struct{}) []ParsedMessageEntity {
	if m == nil {
		return nil
	}
	var out []ParsedMessageEntity
	add := func(text string, entities []MessageEntity) {
		out = append(out, ParseEntityTypes(text, entities, accepted)...)
	}
	addTasks := func(tasks []ChecklistTask) {
		for _, t := range tasks {
			add(t.Text, t.TextEntities)
		}
	}

	add(m.Text, m.Entities)
	add(m.Caption, m.CaptionEntities)
	if c := m.Checklist; c != nil {
		add(c.Title, c.TitleEntities)
		addTasks(c.Tasks)
	}
	if g := m.Game; g != nil {
		add(g.Text, g.TextEntities)
	}
	for _, g := range []*GiftInfo{m.Gift, m.GiftUpgradeSent} {
		if g != nil {
			add(g.Text, g.Entities)
		}
	}
	if g := m.UniqueGift; g != nil {
		add(g.Text, g.Entities)
	}
	if p := m.Poll; p != nil {
		add(p.Question, p.QuestionEntities)
		for _, o := range p.Options {
			add(o.Text, o.TextEntities)
		}
		add(p.Explanation, p.ExplanationEntities)
		add(p.Description, p.DescriptionEntities)
	}
	if o := m.PollOptionAdded; o != nil {
		add(o.OptionText, o.OptionTextEntities)
	}
	if o := m.PollOptionDeleted; o != nil {
		add(o.OptionText, o.OptionTextEntities)
	}
	if t := m.ChecklistTasksAdded; t != nil {
		addTasks(t.Tasks)
	}
	return out
}
