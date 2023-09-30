package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/kvalv/htmx-demo/domain/todo"
)

type TodoHandler struct {
	todos *todo.List
	http.Handler
    t *template.Template
}

func (th *TodoHandler) index(w http.ResponseWriter, r *http.Request) {
    data := struct {
        Todos []*todo.Todo
    }{
        Todos: th.todos.List(),
    }
    fmt.Println("got data..", data)
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

func newHandler() *TodoHandler {
	mux := mux.NewRouter()
	handler := &TodoHandler{
		todos:   todo.NewList(),
		Handler: mux,
        t: template.Must(template.ParseFiles("templates/index.html")),
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

func main() {
	srv := newHandler()
	go func() {
		port := "3000"
		fmt.Println("Listening on port " + port)
		if err := http.ListenAndServe(":"+port, srv); err != nil {
			log.Println(err)
		}
	}()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
	os.Exit(0)
}
