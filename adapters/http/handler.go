package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/kvalv/htmx-demo/domain/todo"
)

type TodoHandler struct {
	http.Handler

	todos *todo.List
	t     *template.Template
}

func (th *TodoHandler) index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Todos []*todo.Todo
	}{
		Todos: th.todos.List(),
	}
	if err := th.t.ExecuteTemplate(w, "index", data); err != nil {
		panic(err)
	}
}
func (th *TodoHandler) list(w http.ResponseWriter, r *http.Request) {

}
func (th *TodoHandler) add(w http.ResponseWriter, r *http.Request) {
	// get form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text := r.FormValue("text")
	th.todos.Add(text)
	data := struct {
		Todos []*todo.Todo
	}{
		Todos: th.todos.List(),
	}
	if err := th.t.ExecuteTemplate(w, "todos", &data); err != nil {
		panic(err)
	}
}
func (th *TodoHandler) reorder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	indices := r.Form["id"]
	fmt.Println("indices", indices, r.Form)
	intIndices := make([]int, len(indices))
	for i, v := range indices {
		intIndices[i], _ = strconv.Atoi(v)
	}
	if len(intIndices) != len(th.todos.List()) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	th.todos.Reorder(intIndices)
	data := struct {
		Todos []*todo.Todo
	}{
		Todos: th.todos.List(),
	}
	if err := th.t.ExecuteTemplate(w, "todos", &data); err != nil {
		panic(err)
	}
}
func (th *TodoHandler) remove(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    v, err := strconv.Atoi(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    th.todos.Remove(v)
    // write null response because the row is removed
}
func (th *TodoHandler) toggle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	v, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	th.todos.Toggle(v)

	if err := th.t.ExecuteTemplate(w, "todo", th.todos.Get(v)); err != nil {
		panic(err)
	}
}

func logmiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func NewTodoHandler() *TodoHandler {
	mux := mux.NewRouter()
	handler := &TodoHandler{
		todos:   todo.NewList(),
		Handler: mux,
		t:       template.Must(template.ParseFiles("templates/index.html")),
	}

	handler.todos.Add("Buy milk")
	handler.todos.Add("Feed cat")
	handler.todos.Add("Drown the cat")

	mux.Use(logmiddleware)
	mux.HandleFunc("/", handler.index).Methods("GET")
	mux.HandleFunc("/todos", handler.list).Methods("GET")
	mux.HandleFunc("/todos", handler.add).Methods("POST")
	mux.HandleFunc("/todos/{id}", handler.remove).Methods("DELETE")
	mux.HandleFunc("/todos/{id}/toggle", handler.toggle).Methods("PUT")
	mux.HandleFunc("/todos/reorder", handler.reorder).Methods("POST")
	return &TodoHandler{
		Handler: mux,
	}
}

func (th *TodoHandler) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return nil
}
