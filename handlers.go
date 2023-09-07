package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.ServeFile(w, r, "tmpl/register.html")
		return
	}
	// grab user info
	username := r.FormValue("username")
	password := r.FormValue("password")
	role := r.FormValue("role")
	// Check existence of user
	var user User
	err := a.db.QueryRow("SELECT username, password, role FROM users WHERE username=$1",
		username).Scan(&user.Username, &user.Password, &user.Role)
	switch {
	// user is available
	case err == sql.ErrNoRows:
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		a.checkInternalServerError(err, w)
		// insert to database
		_, err = a.db.Exec(`INSERT INTO users(username, password, role) VALUES($1, $2, $3)`,
			username, hashedPassword, role)
		a.checkInternalServerError(err, w)
	case err != nil:
		http.Error(w, "loi: "+err.Error(), http.StatusBadRequest)
		return
	default:
		http.Redirect(w, r, "/login", http.StatusMovedPermanently)
	}
}

func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Method %s", r.Method)
	if r.Method != "POST" {
		http.ServeFile(w, r, "tmpl/login.html")
		return
	}
	// grab user info from the submitted form
	username := r.FormValue("usrname")
	password := r.FormValue("psw")

	// query database to get match username
	var user User
	err := a.db.QueryRow("SELECT username, password FROM users WHERE username=$1",
		username).Scan(&user.Username, &user.Password)
	a.checkInternalServerError(err, w)

	// validate password
	/*
		//simple unencrypted method
		if user.Password != password {
			http.Redirect(w, r, "/login", 301)
			return
		}
	*/

	//password is encrypted
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		http.Redirect(w, r, "/login", 301)
		return
	}

	a.authenticated = true
	http.Redirect(w, r, "/list", 301)
	log.Printf("login good")
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	a.authenticated = false
	a.isAuthenticated(w, r)
}

func (a *App) listHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	}
	rows, err := a.db.Query("SELECT * FROM cost")
	a.checkInternalServerError(err, w)
	var funcMap = template.FuncMap{
		"multiplication": func(n int, f int) int {
			return n * f
		},
		"addOne": func(n int) int {
			return n + 1
		},
	}
	var costs []Cost
	var cost Cost
	for rows.Next() {
		err = rows.Scan(&cost.Id, &cost.ElectricAmount,
			&cost.ElectricPrice, &cost.WaterAmount, &cost.WaterPrice, &cost.CheckedDate)
		a.checkInternalServerError(err, w)
		costs = append(costs, cost)
	}
	t, err := template.New("list.html").Funcs(funcMap).ParseFiles("tmpl/list.html")
	a.checkInternalServerError(err, w)
	err = t.Execute(w, costs)
	a.checkInternalServerError(err, w)

}

func (a *App) createHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	if r.Method != "POST" {
		http.Redirect(w, r, "/", 301)
	}
	var cost Cost
	cost.ElectricAmount, _ = strconv.Atoi(r.FormValue("ElectricAmount"))
	cost.ElectricPrice, _ = strconv.Atoi(r.FormValue("ElectricPrice"))
	cost.WaterAmount, _ = strconv.Atoi(r.FormValue("WaterAmount"))
	cost.WaterPrice, _ = strconv.Atoi(r.FormValue("WaterPrice"))
	cost.CheckedDate = r.FormValue("CheckedDate")
	fmt.Println(cost)

	// Save to database
	stmt, err := a.db.Prepare(`
		INSERT INTO cost(electric_amount, electric_price, water_amount, water_price, checked_date)
		VALUES($1, $2, $3, $4, $5)
	`)
	if err != nil {
		fmt.Println("Prepare query error")
		panic(err)
	}
	_, err = stmt.Exec(cost.ElectricAmount, cost.ElectricPrice,
		cost.WaterAmount, cost.WaterPrice, cost.CheckedDate)
	if err != nil {
		fmt.Println("Execute query error")
		panic(err)
	}
	http.Redirect(w, r, "/", 301)
}

func (a *App) updateHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	if r.Method != "POST" {
		http.Redirect(w, r, "/", 301)
	}
	var cost Cost
	cost.Id, _ = strconv.Atoi(r.FormValue("Id"))
	cost.ElectricAmount, _ = strconv.Atoi(r.FormValue("ElectricAmount"))
	cost.ElectricPrice, _ = strconv.Atoi(r.FormValue("ElectricPrice"))
	cost.WaterAmount, _ = strconv.Atoi(r.FormValue("WaterAmount"))
	cost.WaterPrice, _ = strconv.Atoi(r.FormValue("WaterPrice"))
	cost.CheckedDate = r.FormValue("CheckedDate")
	stmt, err := a.db.Prepare(`
		UPDATE cost SET electric_amount=$1, electric_price=$2, water_amount=$3, water_price=$4, checked_date=$5
		WHERE id=$6
	`)
	a.checkInternalServerError(err, w)
	res, err := stmt.Exec(cost.ElectricAmount, cost.ElectricPrice,
		cost.WaterAmount, cost.WaterPrice, cost.CheckedDate, cost.Id)
	a.checkInternalServerError(err, w)
	_, err = res.RowsAffected()
	a.checkInternalServerError(err, w)
	http.Redirect(w, r, "/", 301)
}

func (a *App) deleteHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	if r.Method != "POST" {
		http.Redirect(w, r, "/", 301)
	}
	var costId, _ = strconv.ParseInt(r.FormValue("Id"), 10, 64)
	stmt, err := a.db.Prepare("DELETE FROM cost WHERE id=$1")
	a.checkInternalServerError(err, w)
	res, err := stmt.Exec(costId)
	a.checkInternalServerError(err, w)
	_, err = res.RowsAffected()
	a.checkInternalServerError(err, w)
	http.Redirect(w, r, "/", 301)

}

func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	http.Redirect(w, r, "/list", 301)
}
