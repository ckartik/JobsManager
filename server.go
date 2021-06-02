package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ckaritk/JobsManager/Jobs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Env struct {
	jm      Jobs.JobsManager
	counter int
}

func main() {
	r := chi.NewRouter()

	// Dependency Injection.
	env := &Env{}

	// A good base middleware stack
	/*
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
	*/

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	})

	// RESTy routes for "Jobs" resource
	r.Route("/Job", func(r chi.Router) {
		r.Post("/", env.createJob)
		r.Get("/", env.createJob)
		/*
			  // Subrouters:
			  r.Route("/{jobID}", func(r chi.Router) {
				r.Use(env.JobsCtx)
				r.Get("/", getJob)
			  })
		*/
	})

	log.Println("Starting server at 8443")
	http.ListenAndServe(":8443", r)
}

/*
func (env * Env) JobsCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobID")
		job, ok := env.jm.Jobs.Load(jobID)
		if !ok {
		  http.Error(w, http.StatusText(422), 422)
		  return
		}
		ctx := context.WithValue(r.Context(), "job", job)
		next.ServeHTTP(w, r.WithContext(ctx))
	  })
}
*/

/*

// TODO: @ckartik add authed info to request context.
 func (env *Env) createJob(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	// article, ok := ctx.Value("job").(*exec.Cmd)
	/*
	if !ok {
	  http.Error(w, http.StatusText(422), 422)
	  return
	}
	w.Write([]byte(fmt.Sprintf("title:")))

*/

// Show proof of access to env var.
// TODO: @ckartik add authed info to request context.
func (env *Env) createJob(w http.ResponseWriter, r *http.Request) {
	// id, ok := env.jm.Start()
	go func(env *Env) {
		for {
			env.counter += 1
		}
	}(env)
	w.Write([]byte(fmt.Sprintf("Counter is at %v", env.counter)))
}
