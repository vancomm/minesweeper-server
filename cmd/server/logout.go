package main

import "net/http"

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	app.cookies.Clear(w)
}
