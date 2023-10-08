package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
	"github.com/kvalv/htmx-demo/domain/todo"
)

type TodoHandler struct {
	http.Handler

	todos   *todo.List
	t       *template.Template
	changes chan todo.Todo
	ctx     context.Context
	el      *eventListener
}

type event struct {
	name    string
	payload *string
}
type subscriber struct {
	c chan *event
}
type eventListener struct {
	subs []subscriber
}

func (el *eventListener) add() (*subscriber, func()) {
	s := subscriber{
		c: make(chan *event),
	}
	el.subs = append(el.subs, s)
	closer := func() {
		close(s.c)
		// and remove from subs
		for i, sub := range el.subs {
			if sub.c == s.c {
				el.subs = append(el.subs[:i], el.subs[i+1:]...)
				return
			}
		}
	}
	return &s, closer
}
func (el *eventListener) notify(msg *event) {
	log.Info().Int("numSubs", len(el.subs)).Msg("notifying")
	for _, sub := range el.subs {
		sub.c <- msg
	}
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
	t := th.todos.Get(v)
	log.Info().Str("text", t.Text).Bool("done", t.Done).Msg("toggled")
	th.changes <- *t
	if err := th.t.ExecuteTemplate(w, "todo", t); err != nil {
		panic(err)
	}
}
func (th *TodoHandler) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.(http.Flusher).Flush()
	log.Info().Msg("event source initiated")
	ch, close := th.el.add()
	defer close()

	// send initial state
	w.(http.Flusher).Flush()
	t0 := time.Now()
	for {
		select {
		case <-r.Context().Done():
			log.Info().Msg("subscriber closed afer " + time.Since(t0).String())
			return
		case <-th.ctx.Done():
			log.Info().Msg("context closed")
			return
		case ev := <-ch.c:
			log.Info().Msg("sending event")
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, *ev.payload)
			w.(http.Flusher).Flush()
		}
	}
}
func logmiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info().Str("method", r.Method).Str("url", r.URL.String()).Msg("request")
		next.ServeHTTP(w, r)
	})
}

func NewTodoHandler(ctx context.Context) *TodoHandler {
	mux := mux.NewRouter()
	th := &TodoHandler{
		Handler: mux,
		todos:   todo.NewList(),
		t:       template.Must(template.ParseFiles("templates/index.html")),
		changes: make(chan todo.Todo, 10),
		ctx:     ctx,
		el: &eventListener{
			subs: make([]subscriber, 0),
		},
	}

	th.todos.Add("Buy milk")
	th.todos.Add("Feed cat")
	th.todos.Add("Drown the cat")

	mux.Use(logmiddleware)
	mux.HandleFunc("/", th.index).Methods("GET")
	mux.HandleFunc("/todos", th.list).Methods("GET")
	mux.HandleFunc("/todos", th.add).Methods("POST")
	mux.HandleFunc("/todos/{id}", th.remove).Methods("DELETE")
	mux.HandleFunc("/todos/{id}/toggle", th.toggle).Methods("PUT")
	mux.HandleFunc("/todos/reorder", th.reorder).Methods("POST")
	mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/events", th.events).Methods("GET")

	go func() {
		for {
			select {
			case <-th.ctx.Done():
				log.Info().Msg("done")
				return
			case t := <-th.changes:
				s := strings.Builder{}
				th.t.ExecuteTemplate(&s, "todo", t)
				data := strings.ReplaceAll(s.String(), "\n", "")
				ev := event{
					name:    fmt.Sprintf("change-%d", t.ID),
					payload: &data,
				}
				th.el.notify(&ev)
			}
		}
	}()

	return &TodoHandler{
		Handler: mux,
	}
}

func (th *TodoHandler) Shutdown() error {
	log.Info().Msg("Shutting down server...")
	return nil
}
