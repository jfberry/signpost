package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

var pokemonTemplate *template.Template
var pokestopTemplate *template.Template
var gymTemplate *template.Template

func main() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	config = cfg

	//	templateStr := "https://maps.google.com/maps?q={{.lat}},{{.lon}}"

	pokemonTemplate = template.New("pokemon")
	pokestopTemplate = template.New("pokestop")
	gymTemplate = template.New("gym")

	for _, t := range config.Pokemon {
		pokemonTemplate, err = pokemonTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	for _, t := range config.Pokestop {
		pokestopTemplate, err = pokestopTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	for _, t := range config.Gym {
		gymTemplate, err = gymTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/pokemon/:pokemon_id/:template", GetPokemon)
	r.GET("/pokestop/:pokestop_id/:template", GetPokestop)
	r.GET("/gym/:gym_id/:template", GetGym)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: r,
	}
	fmt.Printf("%s [] Starting server on port %d\n", time.Now().Format(config.TimestampFormat), config.Port)
	srv.ListenAndServe()
}
