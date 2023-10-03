package todo

type Todo struct {
    ID   int
	Text string
	Done bool
}

func New(id int, test string) *Todo {
	return &Todo{
        ID: id,
		Text: test,
		Done: false,
	}
}

func (t *Todo) Toggle() {
	t.Done = !t.Done
}
