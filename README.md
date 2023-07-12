# go-struct2serve
This is a small library that has a generic repository for accessing the database and populating structures in golang and has a generic controller and service.

You can create a custom repository or custom controller using Go compose.


## Example 

#### user_model.go
``` go
package models

type User struct {
	ID        int    `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"first_name"`
	Email     string `json:"email" db:"email"`
	Roles     []Role `json:"roles" sql:"select * from roles r where r.id in (select role_id from user_roles where user_id = ?)"`
}
```

#### role_model.go
``` go
package models

type Role struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}
```

#### project_model.go
``` go
package models

import "time"

type Project struct {
	ID            int        `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	StartProject  *time.Time `json:"start_project" db:"start_project"`
	UserCreatedId int        `json:"user_created_id" db:"user_created_id"`
	UserCreated   User       `json:"user_created" sql:"select id, first_name, email from user where id = ?"` // calculado
	Comments      []Comment  `json:"comments" sql:"select * from comment where comment_id is NULL and project_id = ?"`
}
```

#### project_model.go
``` go
package models

type Epic struct {
	ID             int       `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	UserCreatedId  int       `json:"user_created_id" db:"user_created_id"`
	UserCreated    User      `json:"user_created" sql:"select * from user where id = ?" sql_param:"UserCreatedId"`
	UserAssignedId *int      `json:"user_assigned_id" db:"user_assigned_id"`
	UserAssigned   *User     `json:"user_assigned" sql:"select * from user where id = ?" sql_param:"UserAssignedId"`
	Description    string    `json:"description" db:"description"`
	ProjectId      int       `json:"project_id" db:"project_id"`
	Comments       []Comment `json:"comments" sql:"select * from comment where comment_id is NULL and epic_id = ?" `
}
```

#### comment_model.go
``` go
package models

import "time"

type Comment struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	UserId    int       `json:"user_id" db:"user_id"`
	ProjectId int       `json:"project_id" db:"project_id"`
	EpicId    *int      `json:"epic_id" db:"epic_id"`
	StoryId   *int      `json:"story_id" db:"story_id"`
	TaskId    *int      `json:"task_id" db:"task_id"`
	CommentId *int      `json:"comment_id" db:"comment_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Comments  []Comment `json:"comments"  sql:"select * from comment where comment_id = ?"`
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