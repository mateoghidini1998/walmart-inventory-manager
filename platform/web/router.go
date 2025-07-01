package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() *Router {
	return &Router{
		routerGroup: RouterGroup{
			rt: chi.NewRouter(),
		},
	}
}

type Router struct {
	routerGroup RouterGroup
}

func (r *Router) Use(md ...func(HandleFunc) HandleFunc) {
	r.routerGroup.Use(md...)
}

func (r *Router) Handle(method string, path string, hd HandleFunc, md ...func(HandleFunc) HandleFunc) {
	r.routerGroup.Handle(method, path, hd, md...)
}

func (r *Router) Route(path string, fn func(rg *RouterGroup)) {
	r.routerGroup.Route(path, fn)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request){
	r.routerGroup.rt.ServeHTTP(w, req)
}

func (r *Router) Run(addr string) (err error) {
	err = http.ListenAndServe(addr, r)
	return
}