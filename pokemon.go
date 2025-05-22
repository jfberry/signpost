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

	clientIP := c.GetHeader("CF-Connecting-IP")
	// fallback if someone is not using cloudflare
	if clientIP == "" {
		clientIP = c.ClientIP()
	}

	var pokemonRecord any
	err := getJson(fmt.Sprintf("%s/api/pokemon/id/%s", config.Golbat.Url, pokemonId), &pokemonRecord)

	if pokemonRecord == nil {
		msg := "Unable to get location, the pokemon might have despawned\n"
		fmt.Printf("%s [%s] %s", time.Now().Format(config.TimestampFormat), clientIP, msg)
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", []byte(msg))

	} else {
		var doc bytes.Buffer
		err = pokemonTemplate.ExecuteTemplate(&doc, template, pokemonRecord)
		_ = err
		s := doc.String()
		fmt.Printf("%s [%s] Redirecting to %s\n", time.Now().Format(config.TimestampFormat), clientIP, s)
		c.Redirect(http.StatusMovedPermanently, s)
	}
}

func GetPokestop(c *gin.Context) {
	pokestopId := c.Param("pokestop_id")
	template := c.Param("template")

	clientIP := c.GetHeader("CF-Connecting-IP")
	// fallback if someone is not using cloudflare
	if clientIP == "" {
		clientIP = c.ClientIP()
	}

	var pokestopRecord any
	err := getJson(fmt.Sprintf("%s/api/pokestop/id/%s", config.Golbat.Url, pokestopId), &pokestopRecord)

	if pokestopRecord == nil {
		msg := "Unable to get location, the pokestop might not exist\n"
		fmt.Printf("%s [%s] %s", time.Now().Format(config.TimestampFormat), clientIP, msg)
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", []byte(msg))

	} else {
		var doc bytes.Buffer
		err = pokemonTemplate.ExecuteTemplate(&doc, template, pokestopRecord)
		_ = err
		s := doc.String()
		fmt.Printf("%s [%s] Redirecting to %s\n", time.Now().Format(config.TimestampFormat), clientIP, s)
		c.Redirect(http.StatusMovedPermanently, s)
	}
}
