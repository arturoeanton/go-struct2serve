package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/arturoeanton/go-struct2serve/config"
	_ "github.com/mattn/go-sqlite3"
)

func Hello(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty name")
	}
	message := fmt.Sprintf("Hi, %v. Welcome!", name)
	return message, nil
}

func Config() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		return db, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, first_name TEXT, email TEXT)")
	if err != nil {
		return db, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS roles (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		return db, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS user_roles (id INTEGER PRIMARY KEY, user_id INTEGER, role_id INTEGER)")
	if err != nil {
		return db, err
	}

	//validate if exist users
	var count int
	err = db.QueryRow("SELECT count(*) FROM roles").Scan(&count)
	if err != nil {
		return db, err
	}
	if count == 0 {

		_, err = db.Exec("INSERT INTO user (first_name, email) VALUES ('admin', 'admin@admin.com')")
		if err != nil {
			return db, err
		}
		_, err = db.Exec("INSERT INTO user (first_name, email) VALUES ('user', 'user@user.com')")
		if err != nil {
			return db, err
		}
		_, err = db.Exec("INSERT INTO roles (name) VALUES ('admin')")
		if err != nil {
			return db, err
		}
		_, err = db.Exec("INSERT INTO roles (name) VALUES ('user')")
		if err != nil {
			return db, err
		}
		_, err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (1, 1)")
		if err != nil {
			return db, err
		}
		_, err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (2, 2)")
		if err != nil {
			return db, err
		}
	}
	return db, nil
}

type User struct {
	ID        int    `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"first_name"`
	Email     string `json:"email" db:"email"`
	Roles     []Role `json:"roles" sql:"select * from roles r where r.id in (select role_id from user_roles where user_id = ?)"`
}

type Role struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

func TestGetAll(t *testing.T) {
	config.DB, _ = Config()
	defer config.DB.Close()

	repoUser := NewRepository[User]()
	//repoRole := NewRepositoryWithTable[Role]("roles")

	users, _ := repoUser.GetAll()
	for _, user := range users {
		fmt.Println(user)
	}

}
