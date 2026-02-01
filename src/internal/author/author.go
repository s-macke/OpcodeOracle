package author

import "errors"

var ErrInvalidAuthor = errors.New("invalid author: must be 'user' or 'assistant'")

type Author int

const (
	User Author = iota
	Assistant
)

func (a Author) String() string {
	switch a {
	case User:
		return "user"
	case Assistant:
		return "assistant"
	default:
		return "unknown"
	}
}

func Parse(s string) (Author, error) {
	switch s {
	case "user":
		return User, nil
	case "assistant":
		return Assistant, nil
	default:
		return 0, ErrInvalidAuthor
	}
}
