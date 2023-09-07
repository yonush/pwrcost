package main

import "net/http"

func (a *App) checkInternalServerError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) isAuthenticated(w http.ResponseWriter, r *http.Request) {
	if !a.authenticated {
		http.Redirect(w, r, "/login", 301)
	}
}
