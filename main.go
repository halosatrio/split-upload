package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	r.POST("/upload", func(c *gin.Context) {
		c.JSON(20, gin.H{
			"status":  200,
			"message": "success",
		})
	})

	r.Run(":8080")
}
