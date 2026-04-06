package utils

type Stack[T any] struct {
	Items []T
}

func (s *Stack[T]) Len() int {
	return len(s.Items)
}

func (s *Stack[T]) Push(item T) {
	s.Items = append(s.Items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.Items) == 0 {
		var zero T
		return zero, false
	}

	item := s.Items[len(s.Items)-1]
	s.Items = s.Items[:len(s.Items)-1]
	return item, true
}

func (s *Stack[T]) Peek() *T {
	if len(s.Items) == 0 {
		var zero T
		return &zero
	}

	return &s.Items[len(s.Items)-1]
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.Items) == 0
}
