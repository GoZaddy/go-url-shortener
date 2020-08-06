package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"

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
	linkIDIndexModel := mongo.IndexModel{
		Keys: bson.M{
			"link_id": 1,
		},
		Options: options.Index().SetUnique(true),
	}

	expiresAtIndexModel := mongo.IndexModel{
		Keys: bson.M{
			"expires_at": 1,
		},
		Options: options.Index().SetExpireAfterSeconds(int32(120)),
	}

	linksCol.Indexes().CreateMany(context.Background(), []mongo.IndexModel{linkIDIndexModel, expiresAtIndexModel})
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

		if (url.ExpiresAt != time.Time{}) {
			if time.Now().Before(url.ExpiresAt) {
				fmt.Println("not expired!")
				c.Redirect(http.StatusTemporaryRedirect, url.OriginalURL)
			} else {
				fmt.Println("expired!")
				c.JSON(http.StatusNotFound, gin.H{"message": "Sorry! This link has expired"})
			}
		} else {
			c.Redirect(http.StatusPermanentRedirect, url.OriginalURL)
		}

	})

	router.POST("/api/shorten", func(c *gin.Context) {
		var randomID string
		var expiresAt time.Time

		var urlInRequest struct {
			Link         string `form:"link" json:"link" binding:"required"`
			Result       string `form:"result" json:"result"`               //optional
			ExpiresAfter string `form:"expires_after" json:"expires_after"` //must be in minutes - optional
		}

		//validation
		if c.Request.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			urlInRequest.Link = c.PostForm("link")
			urlInRequest.Result = c.PostForm("result")
			urlInRequest.ExpiresAfter = c.PostForm("expires_after")

		} else {
			if err := c.ShouldBindJSON(&urlInRequest); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
				return
			}
		}

		if _, err := url.ParseRequestURI(urlInRequest.Link); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid url"})
			return
		}

		fmt.Println(urlInRequest)

		if urlInRequest.Result == "" {
			randomID = generateRandomString()
		} else {
			randomID = urlInRequest.Result
		}

		if urlInRequest.ExpiresAfter == "" {
			expiresAt = time.Time{}
		} else {
			minutes, err := strconv.Atoi(urlInRequest.ExpiresAfter)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid duration"})
			}
			expiresAt = time.Now().Add(time.Duration(int64(minutes)) * time.Minute)
		}

		for {
			res := linksCol.FindOne(context.Background(), bson.M{"link_id": randomID})
			if res.Err() != nil {
				if errors.Is(res.Err(), mongo.ErrNoDocuments) {
					break
				}
				c.JSON(500, gin.H{
					"message": "Internal Server error!",
				})
				return
			}
			if urlInRequest.Result != "" {
				c.JSON(http.StatusConflict, gin.H{"message": "This link already exists! Use another link"})
				return
			}
			randomID = generateRandomString()
		}

		urlModel := models.URL{
			ID:          randomID,
			OriginalURL: urlInRequest.Link,
			ExpiresAt:   expiresAt,
		}

		_, err := linksCol.InsertOne(context.Background(), urlModel)
		if err != nil {
			c.JSON(500, gin.H{
				"message": "Internal Server error: error writing url to DB",
			})
			return
		}

		c.JSON(200, gin.H{
			"url": "http://" + c.Request.Host + "/" + randomID,
		})

	})

	router.Run(":9990")
}
