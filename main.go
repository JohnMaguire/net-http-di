package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/goava/di"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type logrusTracer struct {
	logrus.Ext1FieldLogger
}

func (l *logrusTracer) Trace(format string, args ...interface{}) {
	l.Ext1FieldLogger.WithField("source", "goava/di").Tracef(format, args...)
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	di.SetTracer(&logrusTracer{log})

	// Create an application container
	container, err := di.New(
		di.Provide(func() *CounterRepo {
			return &CounterRepo{}
		}),
	)
	if err != nil {
		panic(err)
	}

	// MakeHandler creates a request-scoped IoC container and wraps net/http-like functions
	// (e.g. `func(w http.ResponseWriter, r *http.Request, s *CounterService)`) executing the
	// handlers with its resolved dependencies.
	MakeHandler := func(
		fn any,
	) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Create a container for the request
			requestContainer, err := di.New(
				di.ProvideValue(w, di.As(new(http.ResponseWriter))),
				di.ProvideValue(r),
				di.Provide(func() *logrus.Entry {
					return log.WithField("requestID", uuid.New())
				}),
				di.Provide(func(l *logrus.Entry, r *CounterRepo) *CounterService {
					return &CounterService{repo: r, log: l}
				}),
			)
			if err != nil {
				panic(err)
			}
			defer container.Cleanup()
			requestContainer.AddParent(container)

			// Invoke the handler using reflection in the container to resolve
			// its dependencies
			err = requestContainer.Invoke(fn)
			if err != nil {
				panic(err)
			}
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

// CODE BELOW IMPLEMENTS BASIC SERVICE DEPENDENCIES

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
