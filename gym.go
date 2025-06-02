package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetGym(c *gin.Context) {
	gymId := c.Param("gym_id")
	template := c.Param("template")

	clientIP := c.GetHeader("CF-Connecting-IP")
	// fallback if someone is not using cloudflare
	if clientIP == "" {
		clientIP = c.ClientIP()
	}

	var gymRecord any
	err := getJson(fmt.Sprintf("%s/api/gym/id/%s", config.Golbat.Url, gymId), &gymRecord)

	if gymRecord == nil {
		msg := "Unable to get location, the gym might not exist\n"
		fmt.Printf("%s [%s] %s", time.Now().Format(config.TimestampFormat), clientIP, msg)
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", []byte(msg))

	} else {
		var doc bytes.Buffer
		err = gymTemplate.ExecuteTemplate(&doc, template, gymRecord)
		_ = err
		s := doc.String()
		fmt.Printf("%s [%s] Redirecting to %s\n", time.Now().Format(config.TimestampFormat), clientIP, s)
		c.Redirect(http.StatusMovedPermanently, s)
	}
}
