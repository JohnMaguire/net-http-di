package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sarulabs/di/v2"
	"github.com/sirupsen/logrus"
)

type logrusTracer struct {
	logrus.Ext1FieldLogger
}

func (l *logrusTracer) Trace(format string, args ...interface{}) {
	l.Ext1FieldLogger.WithField("source", "sarulabs/di").Tracef(format, args...)
}

func main() {
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)

	// Create dependency container and add dependency providers
	builder, err := di.NewBuilder()
	if err != nil {
		panic(err)
	}

	// App dependency scope
	err = builder.Add(di.Def{
		Name: "CounterRepo",
		Build: func(ctn di.Container) (interface{}, error) {
			return &CounterRepo{}, nil
		},
	})
	if err != nil {
		panic(err)
	}

	// Request dependency scope
	err = builder.Add(di.Def{
		Name: "logrusEntry",
		Build: func(ctn di.Container) (interface{}, error) {
			l := log.WithField("requestID", uuid.New())
			return l, nil
		},
		Scope: di.Request, // XXX
	})
	if err != nil {
		panic(err)
	}

	err = builder.Add(di.Def{
		Name: "CounterService",
		Build: func(ctn di.Container) (interface{}, error) {
			r, err := ctn.SafeGet("CounterRepo")
			if err != nil {
				return nil, err
			}

			l, err := ctn.SafeGet("logrusEntry")
			if err != nil {
				return nil, err
			}

			return &CounterService{
				repo: r.(*CounterRepo),
				log:  l.(*logrus.Entry),
			}, nil
		},
		Scope: di.Request, // XXX
	})
	if err != nil {
		panic(err)
	}

	appContainer := builder.Build()

	// MakeHandler creates a request-scoped container and wraps net/http-like functions
	// (e.g. `func(w http.ResponseWriter, r *http.Request, c di.Container)`) executing the
	// handlers with the container (service locator) for it to fetch its dependencies from.
	MakeHandler := func(
		fn func(w http.ResponseWriter, r *http.Request, c di.Container),
	) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Create a request container
			requestContainer, err := appContainer.SubContainer()
			if err != nil {
				panic(err)
			}
			defer requestContainer.Delete()

			fn(w, r, requestContainer)
		}
	}

	r := chi.NewRouter()
	r.Get("/", MakeHandler(func(w http.ResponseWriter, r *http.Request, cnt di.Container) {
		// Get the dependency from the container
		s := cnt.Get("CounterService").(*CounterService)

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
