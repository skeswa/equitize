package main

import (
	"github.com/go-martini/martini"
	"log"
)

func main() {
	// Read environment variables
	env, err := NewEnvironment()
	if err != nil {
		log.Fatalln(ERR_COULDNT_START + err.Error())
	}
	// Setup the database
	db, err := SetupDatabase(env)
	if err != nil {
		log.Fatalln(ERR_COULDNT_START + err.Error())
	}
	// Setup the Stripe API
	SetupStripe(env)
	// Setup server
	m := martini.Classic()
	SetupMiddleware(m, db, env)
	SetupRoutes(m, db, env)
	// Start the server
	m.Run()
}
