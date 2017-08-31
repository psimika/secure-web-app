package web

import (
	"net/http"
)

func (s *server) demoXSS(w http.ResponseWriter, r *http.Request) *Error {
	v := r.PostFormValue("xss")
	return s.render(w, r, s.templates.demoXSS, v, nil)
}
