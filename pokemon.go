package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetPokemon(c *gin.Context) {
	pokemonId := c.Param("pokemon_id")
	template := c.Param("template")

	var pokemonRecord any
	err := getJson(fmt.Sprintf("%s/api/pokemon/id/%s", config.Golbat.Url, pokemonId), &pokemonRecord)

	clientIP := c.GetHeader("CF-Connecting-IP")
	// fallback if someone is not using cloudflare
	if clientIP == "" {
		clientIP = c.ClientIP()
	}

	if pokemonRecord == nil {
		msg := "Unable to get location, the pokemon might have despawned\n"
		fmt.Printf("%s [%s] %s", clientIP, time.Now().Format(config.TimestampFormat), msg)
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", []byte(msg))

	} else {
		var doc bytes.Buffer
		err = pokemonTemplate.ExecuteTemplate(&doc, template, pokemonRecord)
		_ = err
		s := doc.String()
		fmt.Printf("%s [%s] Redirecting to %s\n", clientIP, time.Now().Format(config.TimestampFormat), s)
		c.Redirect(http.StatusMovedPermanently, s)
	}
}
