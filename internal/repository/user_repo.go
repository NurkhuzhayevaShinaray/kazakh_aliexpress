package repository

import (
	"context"
	"errors"
	"time"

	"kazakh_aliexpress/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	Collection *mongo.Collection
}

func (m *UserRepository) Insert(email, password, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.Collection.InsertOne(ctx, user)
	return err
}

func (m *UserRepository) Authenticate(email, password string) (models.User, error) {
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.Collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.User{}, errors.New("invalid credentials")
		}
		return models.User{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return models.User{}, errors.New("invalid credentials")
		}
		return models.User{}, err
	}

	return user, nil
}
