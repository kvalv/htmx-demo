package todo

type List struct {
	list []*Todo
}

func NewList() *List {
	return &List{}
}
func (t *List) List() []*Todo {
	return t.list
}
func (t *List) Reorder(indices []int) {
	newList := make([]*Todo, len(indices))
	for i, v := range indices {
		value := t.Get(v)
		newList[i] = value
	}
	t.list = newList
}
func (t *List) Get(id int) *Todo {
	for _, v := range t.list {
		if v.ID == id {
			return v
		}
	}
	return nil
}
func (t *List) Add(text string) {
	id := len(t.list)
	t.list = append(t.list, New(id, text))
}

func (t *List) Remove(index int) {
	var newList []*Todo
	for _, v := range t.list {
		if v.ID != index {
			newList = append(newList, v)
		}
	}
	t.list = newList
}

func (t *List) Toggle(id int) {
	t.Get(id).Toggle()
}
