package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"metoda/internal/app/auth"
	"metoda/internal/app/ds"
	"metoda/internal/app/serializer"
)

func (r *Repository) GetUserByID(id int) (ds.Users, error) {
	if id <= 0 {
		return ds.Users{}, fmt.Errorf("неверный id: должен быть > 0")
	}
	var u ds.Users
	err := r.db.Where("id = ?", id).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.Users{}, fmt.Errorf("%w: пользователь с id %d", ErrNotFound, id)
		}
		return ds.Users{}, err
	}
	return u, nil
}

func (r *Repository) GetUserByLogin(login string) (ds.Users, error) {
	if login == "" {
		return ds.Users{}, fmt.Errorf("логин не может быть пустым")
	}
	var u ds.Users
	err := r.db.Where("login = ?", login).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.Users{}, fmt.Errorf("%w: пользователь с логином %s", ErrNotFound, login)
		}
		return ds.Users{}, err
	}
	return u, nil
}

func (r *Repository) CreateUser(j serializer.UserJSON) (ds.Users, error) {
	if j.Login == "" {
		return ds.Users{}, fmt.Errorf("логин обязателен")
	}
	if j.Password == "" {
		return ds.Users{}, fmt.Errorf("пароль обязателен")
	}

	_, err := r.GetUserByLogin(j.Login)
	if err == nil {
		return ds.Users{}, fmt.Errorf("%w: пользователь с логином %s уже существует", ErrAlreadyExists, j.Login)
	} else if !errors.Is(err, ErrNotFound) {
		return ds.Users{}, err
	}

	u := serializer.UserFromJSON(j)
	if err := r.db.Create(&u).Error; err != nil {
		return ds.Users{}, fmt.Errorf("ошибка при создании пользователя: %w", err)
	}
	return u, nil
}

func (r *Repository) SignIn(j serializer.UserJSON) (ds.Users, error) {
	if j.Login == "" {
		return ds.Users{}, fmt.Errorf("логин обязателен")
	}
	if j.Password == "" {
		return ds.Users{}, fmt.Errorf("пароль обязателен")
	}

	u, err := r.GetUserByLogin(j.Login)
	if err != nil {
		return ds.Users{}, fmt.Errorf("неверный логин или пароль")
	}
	if u.Password != j.Password {
		return ds.Users{}, fmt.Errorf("неверный логин или пароль")
	}

	r.SetUserID(int(u.ID))
	return u, nil
}

// SignInWithToken проверяет логин/пароль и возвращает пользователя и JWT (лаб. 4).
func (r *Repository) SignInWithToken(j serializer.UserJSON) (ds.Users, string, error) {
	u, err := r.SignIn(j)
	if err != nil {
		return ds.Users{}, "", err
	}
	tok, err := auth.GenerateToken(u.ID, u.IsModerator)
	if err != nil {
		return ds.Users{}, "", err
	}
	return u, tok, nil
}
