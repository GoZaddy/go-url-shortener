package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoDBClient *mongo.Client
	linksCol      *mongo.Collection
)

func connect() {
	mongodbURI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(mongodbURI)
	mongoDBClient, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		log.Fatal(err)
	}
	linksCol = mongoDBClient.Database("Shortener").Collection("links")
	err = mongoDBClient.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected to mongodb")
}

func disconnnect() {
	err := mongoDBClient.Disconnect(context.TODO())

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection to MongoDB closed.")
}
