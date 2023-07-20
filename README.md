# go-struct2serve
This is a small library for [echo v4](https://github.com/labstack/echo) that has a generic repository for accessing the database and populating structures in golang and has a generic controller and service.

You can create a custom repository or custom controller using Go compose.


## Question

* Why Echo? Becouse It is my favorite framwork for make web apps :P.
* Why SQL? **I love SQL**.
* Can you add feature? **yes**, I love you ;).
* Why did I make this library? Because I have ***free time***, but it doesn't mean I don't love GORM :).
* Why make this project? This project is a hobby and We use it in some minor projects. **This is fun**.


## Repository
This repository provides a generic implementation of repository patterns in Go using SQL databases. It allows you to write less code and easily perform CRUD operations. The library supports deep object graph navigation and loading, thanks to tags that can specify how to load the data.

## Features

* Full CRUD operations
* Deep object graph navigation
* Transaction support
* Custom SQL queries through tags
* Custom loading of nested structures

## How It Works

This library uses the reflect package to inspect your data models at runtime. It uses the db tag to map the struct fields to the database columns and it uses custom s2s tags to define how to load the data.

For example, the tag s2s:"id in (select role_id from user_roles where user_id = ?)" tells the library to execute this SQL query to load the roles for a user. The ? placeholder will be replaced with the ID of the user.

You can also specify how to load nested structures using the s2s tag. For example, s2s:"id = ?" s2s_param:"GroupId" tells the library to load the group for a user using the GroupId value.


## Example 

#### user_model.go
``` go
package models

type User struct {
	UserID    int     `json:"id" db:"id" s2s_id:"true"` // mark this field as id with tag s2s_id:"true"
	FirstName string  `json:"first_name" db:"first_name"`
	Email     string  `json:"email" db:"email"`
	Roles     *[]Role `json:"roles,omitempty" s2s:"id in (select role_id from user_roles where user_id = ?)"` // not use s2s_param becuase s2s_param is the id of Struct
	GroupId   *int    `json:"-" db:"group_id" s2s_ref_value:"MyGroup.ID"`                                    // mark this field as id with tag s2s_ref_value:"Group.ID" because json not send nil values json:"-"
	MyGroup     *Group  `json:"group,omitempty" s2s:"id = ?" s2s_param:"GroupId"`                               // use s2s_param becuase we need use GroupId value
	//other way is  MyGroup *Group `json:"group,omitempty" s2s:"select * from groups where id = ?" sql_param:"GroupId"`
}

```

#### role_model.go
``` go
package models

type Role struct {
	ID    int     `json:"id" db:"id" s2s_table_name:"roles"` // use s2s_table_name:"roles" because table name is not the same as struct name
	Name  string  `json:"name" db:"name"`
	Users *[]User `json:"users,omitempty" s2s:"id in (select user_id from user_roles where role_id = ?)"` // not use s2s_param becuase s2s_param is the id of Struct
}
```

#### group_model.go
``` go
package models

type Group struct {
	ID    int     `json:"id" db:"id" s2s_table_name:"groups"` // use s2s_table_name:"groups" because table name is not the same as struct name
	Name  string  `json:"name" db:"name"`
	Users *[]User `json:"users,omitempty" s2s:"group_id = ?"` // not use s2s_param becuase s2s_param is the id of Struct
}
```

## Example custom **repository**

#### user_repositories.go

* The project is "springhub" and the folder models is springhub/models

``` go
package customs

import (
	"context"
	"fmt"
	"log"

	"github.com/arturoeanton/go-struct2serve/config"
	"github.com/arturoeanton/go-struct2serve/repositories"
	"github.com/arturoeanton/springhub/models"
)

type UserRepository struct {
	repositories.Repository[models.User]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		Repository: *repositories.NewRepository[models.User](),
	}
}

func (ur *UserRepository) GetAll() ([]*models.User, error) {
	fmt.Println("Custom GetAll")
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	query := "SELECT * FROM user"
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		log.Printf("Error al ejecutar la consulta: %v", err)
		return nil, err
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(&user.ID, &user.FirstName, &user.Email)
		if err != nil {
			log.Printf("Error al escanear la fila: %v", err)
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
```



## Example custom **handler**


#### project_handlers.go

* The project is "springhub" and the folder models is springhub/models

``` go
package customs

import (
	"net/http"

	"github.com/arturoeanton/go-struct2serve/handlers"
	"github.com/arturoeanton/go-struct2serve/repositories"
	"github.com/arturoeanton/go-struct2serve/services"
	"github.com/arturoeanton/springhub/models"
	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	*handlers.Handler[models.Project]
	projectService services.IService[models.Project]
}

func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		Handler: handlers.NewHandler[models.Project](),
		projectService: services.NewService[models.Project](
			repositories.NewRepository[models.Project](),
		),
	}
}

func (uh *ProjectHandler) FilterByNameOrDesciption(c echo.Context) error {
	name := c.QueryParam("name")
	description := c.QueryParam("description")

	projects, err := uh.projectService.GetByCriteria("name like ? or description  like ? ", "%"+name+"%", "%"+description+"%")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get projects",
		})
	}
	return c.JSON(http.StatusOK, projects)
}
```



## Transactions

The library also supports transactions. You can create a new transaction and set it on your repositories:

```go
tx, _ := CreateTxAndSet(repoUser, repoRole)
```
Then, you can use the Commit() and Rollback() methods to control the transaction:

```go
err := repoUser.Commit()
if err != nil {
	// handle error
}

err = repoUser.Rollback()
if err != nil {
	// handle error
}
```

# Custom SQL Queries

You can execute custom SQL queries using the **GetByCriteria()** method. This method takes a SQL query string and any number of arguments for the query parameters:

```go
users, _ := repoUser.GetByCriteria("first_name = ? AND email = ?", "admin", "admin@admin.com")
```


## Installation

Use the go get command to install this library:

```sh
go get github.com/arturoeanton/go-struct2serve
```

## Tests

This library includes a test suite. You can run the tests using the go test command:

```sh
go test ./...
```