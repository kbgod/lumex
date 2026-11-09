package lumex

// Deprecated aliases for generated types that the Bot API renamed. They keep
// code written against an older lumex release compiling; new code should use
// the current name.

// StoryAreaPosition is the former name of [AreaPosition].
//
// Deprecated: Bot API 10.4 renamed StoryAreaPosition to AreaPosition, because
// the type now also describes the position of stickers on a TON wallet card.
// Use [AreaPosition].
type StoryAreaPosition = AreaPosition
