package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type HandleFunc func(w http.ResponseWriter, r *http.Request) (err error)

type RouterGroup struct {
	rt *chi.Mux
	md []func(HandleFunc) HandleFunc
	basePath string
}

func (rg *RouterGroup) Use(md ...func(HandleFunc) HandleFunc) {
	(*rg).md = append((*rg).md, md...)
}

func (rg *RouterGroup) Handle(method string, path string, hd HandleFunc, md ...func(HandleFunc) HandleFunc) {
	hd = handlerChain(hd, md...)
	hd = handlerChain(hd, (*rg).md...)

	handler := handlerAdapter(hd)
	path = (*rg).basePath + path

	(*rg).rt.Method(method, path, handler)
}

func (rg *RouterGroup) Route(path string, fn func(rg *RouterGroup)) {
	subRouter := &RouterGroup{
		rt: (*rg).rt,
		md: (*rg).md,
		basePath: (*rg).basePath + path,
	}

	fn(subRouter)
}

func handlerChain(hd HandleFunc, md ...func(HandleFunc) HandleFunc) HandleFunc {
	for _, m := range md {
		hd = m(hd)
	}

	return hd
}

func handlerAdapter(hd HandleFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := hd(w, req)
		if err != nil {
			println(err.Error())
		}
	}
}