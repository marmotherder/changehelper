package main

type FatalError struct {
	message string
}

func (m *FatalError) Error() string {
	return m.message
}
