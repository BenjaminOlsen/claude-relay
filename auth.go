package main

import "net/http"

func (s *Server) authenticate(r *http.Request) bool {
	return r.URL.Query().Get("token") == s.token
}
