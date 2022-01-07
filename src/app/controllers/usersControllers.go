package controllers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nmibragimov7/go-app-server/src/app/db"
	"github.com/nmibragimov7/go-app-server/src/app/middleware"
	"github.com/nmibragimov7/go-app-server/src/app/models"
	"github.com/nmibragimov7/go-app-server/src/app/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"strings"
	"time"
)

var UsersCollection = db.OpenCollection(db.OpenDatabase(db.Client, "test"), "users")

const (
	tokenExpiresAt   = time.Hour
	refreshExpiresAt = time.Hour * 3
)

func ParseHeader(c *gin.Context) string {
	tokenHeader := c.GetHeader(middleware.AuthorizationHeaderKey)
	fields := strings.Fields(tokenHeader)
	id := fmt.Sprintf("%v", service.Parse(fields[1]))

	return id
}

func GetProfile(c *gin.Context) {
	id := ParseHeader(c)
	objectId, _ := primitive.ObjectIDFromHex(id)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.User
	err := UsersCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&user)
	userCopy := user.BeforeSend()

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден!"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": userCopy})
}

func GetUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := UsersCollection.Find(ctx, bson.M{})
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objectId, _ := primitive.ObjectIDFromHex(id)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var user models.User
		err := UsersCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&user)
		userCopy := user.BeforeSend()

		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден!"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": userCopy})
	}
}

func SignIn(c *gin.Context) {
	var body models.User

	if err := c.BindJSON(&body); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.User
	err := UsersCollection.FindOne(ctx, bson.M{"username": body.Username}).Decode(&user)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Неверный логин или пароль!"})
		return
	}

	if user.ComparePassword(body.Password) {
		token, errToken := service.JwtCreate(user.ID.Hex(), tokenExpiresAt)
		if errToken != nil {
			fmt.Println(errToken)
			return
		}
		refresh, errRefresh := service.JwtCreate(user.ID.Hex(), refreshExpiresAt)
		if errRefresh != nil {
			fmt.Println(errRefresh)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"token":           token,
			"tokenExpireAt":   time.Now().Add(tokenExpiresAt),
			"refresh":         refresh,
			"refreshExpireAt": time.Now().Add(refreshExpiresAt),
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный логин или пароль!"})
	}
}

func SingUp(c *gin.Context) {
	var body models.User

	if err := c.BindJSON(&body); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"message": "Неверный тип данных",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	encrypt, err := models.EncryptPassword(body.Password)
	if err != nil {
		log.Fatal(err)
	}

	user, err := UsersCollection.InsertOne(ctx, bson.D{
		{"firstName", body.FirstName},
		{"lastName", body.LastName},
		{"username", body.Username},
		{"password", encrypt},
		{"role", "user"},
		{"createdAt", time.Now()},
		{"updatedAt", time.Now()},
	})

	if err != nil {
		log.Fatal(err)
		c.IndentedJSON(http.StatusForbidden, err)
	}
	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"message": "Пользователь успешно зарегистрирован!",
	})
}

func Refresh(c *gin.Context) {
	id := ParseHeader(c)
	objectId, _ := primitive.ObjectIDFromHex(id)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var user models.User
	err := UsersCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&user)

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден!"})
		return
	}

	token, errToken := service.JwtCreate(user.ID.Hex(), tokenExpiresAt)
	if errToken != nil {
		fmt.Println(errToken)
		return
	}
	refresh, errRefresh := service.JwtCreate(user.ID.Hex(), refreshExpiresAt)
	if errRefresh != nil {
		fmt.Println(errRefresh)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":           token,
		"tokenExpireAt":   time.Now().Add(tokenExpiresAt),
		"refresh":         refresh,
		"refreshExpireAt": time.Now().Add(refreshExpiresAt),
	})
}

func EditUser(c *gin.Context) {
	var body models.User

	if err := c.BindJSON(&body); err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"message": "Неверный тип данных!",
		})
		return
	}

	id := c.Param("id")
	objectId, _ := primitive.ObjectIDFromHex(id)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := UsersCollection.UpdateOne(ctx, bson.M{"_id": objectId}, bson.D{{"$set", bson.D{
		{"firstName", body.FirstName},
		{"lastName", body.LastName},
		{"username", body.Username},
		{"role", body.Role},
		{"updatedAt", time.Now()},
	}}})

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Произошла ошибка при обновлении!"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Пользователь обновлен успешно!"})
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	objectId, _ := primitive.ObjectIDFromHex(id)

	headerId := ParseHeader(c)
	objectHeaderId, _ := primitive.ObjectIDFromHex(headerId)

	if objectHeaderId == objectId {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Отказано в удалении!"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := UsersCollection.DeleteOne(ctx, bson.M{"_id": objectId})
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Произошла ошибка при удалении!"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно удален!"})
}