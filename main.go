package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	"time"

	"github.com/gozaddy/go-url-shortener/models"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gin-gonic/gin"
)

var (
	currentLink string = ""
)

func init() {
	godotenv.Load(".env")
	//connect to mongodb
	connect()
	indexModel := mongo.IndexModel{
		Keys: bson.M{
			"link_id": 1,
		},
		Options: options.Index().SetUnique(true),
	}

	linksCol.Indexes().CreateOne(context.Background(), indexModel)
}

func generateRandomString() string {
	rand.Seed(time.Now().UnixNano())
	characters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 6)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("views/*")

	router.GET("/", func(c *gin.Context) {
		fmt.Println(c.Request.Host)
		if currentLink == "" {
			c.HTML(200, "index.gohtml", nil)
		} else {
			c.HTML(200, "index.gohtml", currentLink)
		}

	})

	router.GET("/:linkID", func(c *gin.Context) {
		var url models.URL
		linkID := c.Param("linkID")
		err := linksCol.FindOne(context.Background(), bson.M{"link_id": linkID}).Decode(&url)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(404, gin.H{
					"error": "link not found",
				})
				return
			}
			c.JSON(500, gin.H{
				"error": "Internal server error!",
			})
			return
		}

		c.Redirect(http.StatusPermanentRedirect, url.OriginalURL)
	})

	router.POST("/api/shorten", func(c *gin.Context) {
		var urlInRequest struct {
			Link string `form:"link" json:"link" binding:"required"`
		}

		if c.Request.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			urlInRequest.Link = c.PostForm("link")
		} else {
			if err := c.ShouldBindJSON(&urlInRequest); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
				return
			}
		}
		if _, err := url.ParseRequestURI(urlInRequest.Link); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
			return
		}
		randomID := generateRandomString()

		urlModel := models.URL{
			ID:          randomID,
			OriginalURL: urlInRequest.Link,
		}

		for {
			res := linksCol.FindOne(context.Background(), bson.M{"link_id": randomID})
			if res.Err() != nil {
				if errors.Is(res.Err(), mongo.ErrNoDocuments) {
					break
				}
				c.JSON(500, gin.H{
					"error": "Internal Server error!",
				})
				return
			}
			randomID = generateRandomString()
		}

		_, err := linksCol.InsertOne(context.Background(), urlModel)
		if err != nil {
			c.JSON(500, gin.H{
				"error": "Internal Server error: error writing url to DB",
			})
			return
		}

		/*c.JSON(200, gin.H{
			"url": "http://localhost:4000/" + randomID,
		})*/

		currentLink = "http://" + c.Request.Host + "/" + randomID

		c.Redirect(http.StatusSeeOther, "/")

	})

	router.Run()
}
