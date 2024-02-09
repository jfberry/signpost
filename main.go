package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
)

var pokemonTemplate *template.Template

func main() {
	tomlFile, err := os.Open("config.toml")
	// if we os.Open returns an error then handle it
	if err != nil {
		panic(err)
	}
	// defer the closing of our tomlFile so that we can parse it later on
	defer tomlFile.Close()

	byteValue, _ := io.ReadAll(tomlFile)

	err = toml.Unmarshal(byteValue, &config)
	if err != nil {
		panic(err)
	}
	config.TimestampFormat = "2006-01-02 15:04:05"

	//	templateStr := "https://maps.google.com/maps?q={{.lat}},{{.lon}}"

	pokemonTemplate = template.New("pokemon")

	for _, t := range config.Pokemon {
		pokemonTemplate, err = pokemonTemplate.New(t.Name).Parse(t.Url)
		if err != nil {
			panic(err)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/pokemon/:pokemon_id/:template", GetPokemon)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: r,
	}
	fmt.Println("Starting server on port", config.Port)
	srv.ListenAndServe()
}
