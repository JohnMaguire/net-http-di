package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	// Build global dependencies
	log := logrus.New()
	repo := &CounterRepo{}

	MakeHandler := func(
		// TODO Accept handlers with any dependencies and resolve them
		fn func(w http.ResponseWriter, r *http.Request, s *CounterService),
	) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := &CounterService{
				repo: repo,
				log:  log.WithField("requestID", uuid.New()),
			}

			fn(w, r, s)
		}
	}

	r := chi.NewRouter()
	r.Get("/", MakeHandler(func(w http.ResponseWriter, r *http.Request, s *CounterService) {
		// Use a dependency and render a response
		count := s.IncreaseCount()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("You are visitor #%d", count)))
	}))

	http.ListenAndServe(":3000", r)
}

// CounterService is invoked by a handler and has dependencies of its own.
type CounterService struct {
	repo *CounterRepo
	log  *logrus.Entry
}

func (s *CounterService) IncreaseCount() int {
	s.repo.Add()
	count := s.repo.Get()
	s.log.WithField("count", count).Info("Registered visit")
	return s.repo.Get()
}

// CounterRepo stores the visitor count in memory.
type CounterRepo struct {
	sync.Mutex
	count int
}

func (r *CounterRepo) Add() {
	r.Lock()
	r.count++
	r.Unlock()
}

func (r *CounterRepo) Get() int {
	r.Lock()
	defer r.Unlock()
	return r.count
}
