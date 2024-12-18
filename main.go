package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

var mg MongoInstance

const dbName = "Go_Crud_Mongo"
const mongoURL = "mongodb://localhost:27017/" + dbName

// Movie represents the model for a Movie
type Movie struct {
	ID       string `json:"id,omitempty" bson:"_id,omitempty"`
	Name     string `json:"name"`
	Director string `json:"director"`
	Genre    string `json:"genre"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURL))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		DB:     db,
	}

	return nil

}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	// Get all movies
	app.Get("/movie", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.DB.Collection("movies").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var movies []Movie = make([]Movie, 0)
		if err := cursor.All(c.Context(), &movies); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(movies)
	})

	// Add a new movie
	app.Post("/movie", func(c *fiber.Ctx) error {
		collection := mg.DB.Collection("movies")
		movie := new(Movie)

		if err := c.BodyParser(movie); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		movie.ID = "" // Let MongoDB generate the ID
		insertionResult, err := collection.InsertOne(c.Context(), movie)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		createdMovie := &Movie{}
		createdRecord.Decode(createdMovie)

		return c.Status(201).JSON(createdMovie)
	})

	// Update a movie
	app.Put("/movie/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		movieID, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			return c.SendStatus(400)
		}

		movie := new(Movie)
		if err := c.BodyParser(movie); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: movieID}}
		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: movie.Name},
					{Key: "director", Value: movie.Director},
					{Key: "genre", Value: movie.Genre},
				},
			},
		}

		err = mg.DB.Collection("movies").FindOneAndUpdate(c.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(404)
			}
			return c.SendStatus(500)
		}

		movie.ID = idParam
		return c.Status(200).JSON(movie)
	})

	// Delete a movie
	app.Delete("/movie/:id", func(c *fiber.Ctx) error {
		movieID, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: movieID}}
		result, err := mg.DB.Collection("movies").DeleteOne(c.Context(), query)
		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("Record deleted")
	})

	log.Fatal(app.Listen(":3000"))
}