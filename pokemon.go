package main

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPokemon(c *gin.Context) {
	pokemonId := c.Param("pokemon_id")
	template := c.Param("template")

	var pokemonRecord any
	err := getJson(fmt.Sprintf("%s/api/pokemon/id/%s", config.Golbat.Url, pokemonId), &pokemonRecord)

	var doc bytes.Buffer
	err = pokemonTemplate.ExecuteTemplate(&doc, template, pokemonRecord)
	_ = err
	s := doc.String()
	fmt.Printf("Redirecting to %s\n", s)
	c.Redirect(http.StatusMovedPermanently, s)
}
