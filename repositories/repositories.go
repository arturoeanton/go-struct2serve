package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/arturoeanton/go-struct2serve/config"
	"github.com/arturoeanton/go-struct2serve/utils"
)

type IRepository[T any] interface {
	GetAll() ([]*T, error)
	GetByID(id interface{}) (*T, error)
	GetByCriteria(criteria string, args ...interface{}) ([]*T, error)
	Create(item *T) (int64, error)
	Update(item *T) error
	Delete(id interface{}) error
}

type Repository[T any] struct {
	table            string
	tags             []string
	sqlAll           string
	sqlGetByID       string
	sqlGetByCriteria string
	sqlCreate        string
	sqlUpdate        string
	sqlDelete        string
	tagName          map[string]string
}

func NewRepository[T any]() *Repository[T] {
	item := CreateNewElement[T]()
	itemType := reflect.TypeOf(*item)
	return NewRepositoryWithTable[T](utils.ToSnakeCase(itemType.Name()))
}

func NewRepositoryWithTable[T any](table string) *Repository[T] {

	item := CreateNewElement[T]()

	r := &Repository[T]{
		table: table,
	}

	itemType := reflect.TypeOf(*item)

	r.tagName = make(map[string]string, itemType.NumField())
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}
		r.tags = append(r.tags, tag)
		r.tagName[tag] = field.Name
	}

	sqlCreate := "INSERT INTO " + table + " ("
	sqlUpdate := "UPDATE " + table + " SET "
	for i, field := range r.tags {
		if i > 0 {
			sqlCreate += ", "
			sqlUpdate += ", "
		}
		sqlCreate += field
		sqlUpdate += field + " = ?"
	}
	sqlCreate += ") VALUES ("
	sqlUpdate += " WHERE id = ?"
	for i := 0; i < len(r.tags); i++ {
		if i > 0 {
			sqlCreate += ", "
		}
		sqlCreate += "?"
	}
	sqlCreate += ")"

	fieldList := strings.Join(r.tags, ", ")
	r.sqlAll = "SELECT " + fieldList + " FROM " + table
	r.sqlGetByCriteria = "SELECT " + fieldList + " FROM " + table + " WHERE "
	r.sqlGetByID = "SELECT " + fieldList + " FROM " + table + " WHERE id = ?"
	r.sqlCreate = sqlCreate
	r.sqlUpdate = sqlUpdate
	r.sqlDelete = "DELETE FROM " + table + " WHERE id = ?"

	return r
}

func (r *Repository[T]) GetAll() ([]*T, error) {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), r.sqlAll)
	if err != nil {
		log.Printf("Error al ejecutar la consulta: %v", err)
		return nil, err
	}
	defer rows.Close()
	items := []*T{}
	for rows.Next() {
		item := CreateNewElement[T]()
		v, err := Scan2(reflect.TypeOf(*item), rows)
		//err := Scan[T](item, rows)
		if err != nil {
			log.Printf("Error al escanear la fila: %v", err)
			return nil, err
		}
		items = append(items, v.Addr().Interface().(*T))
	}

	return items, nil
}

func (r *Repository[T]) GetByCriteria(criteria string, args ...interface{}) ([]*T, error) {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), r.sqlGetByCriteria+" "+criteria, args...)
	if err != nil {
		log.Printf("Error al ejecutar la consulta: %v", err)
		return nil, err
	}
	defer rows.Close()
	items := []*T{}
	for rows.Next() {
		item := CreateNewElement[T]()
		v, err := Scan2(reflect.TypeOf(*item), rows)
		//err := Scan[T](item, rows)
		if err != nil {
			log.Printf("Error al escanear la fila: %v", err)
			return nil, err
		}
		items = append(items, v.Addr().Interface().(*T))
	}

	return items, nil
}

func (r *Repository[T]) GetByID(id interface{}) (*T, error) {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	row := conn.QueryRowContext(context.Background(), r.sqlGetByID, id)
	item := CreateNewElement[T]()
	v, err := Scan2(reflect.TypeOf(*item), row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No se encontró el usuario
		}
		log.Printf("Error al escanear la fila: %v", err)
		return nil, err
	}
	return v.Addr().Interface().(*T), nil
}

func (r *Repository[T]) Create(item *T) (int64, error) {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	fieldsValues := []interface{}{}
	for _, tag := range r.tags {
		fieldsValues = append(fieldsValues, reflect.ValueOf(*item).FieldByName(r.tagName[tag]).Interface())
	}

	result, err := conn.ExecContext(context.Background(), r.sqlCreate, fieldsValues...)
	if err != nil {
		log.Printf("Error al crear el item: %v", err)
		return 0, err
	}

	resultID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return resultID, nil
}

func (r *Repository[T]) Update(item *T) error {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	fieldsValues := []interface{}{}
	for _, tag := range r.tags {
		fieldsValues = append(fieldsValues, reflect.ValueOf(item).FieldByName(r.tagName[tag]).Interface())
	}
	fieldsValues = append(fieldsValues, reflect.ValueOf(item).FieldByName("ID").Interface())

	_, err = conn.ExecContext(context.Background(), r.sqlUpdate, fieldsValues...)
	if err != nil {
		log.Printf("Error al actualizar el item: %v", err)
		return err
	}

	return nil
}

func (r *Repository[T]) Delete(id interface{}) error {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.ExecContext(context.Background(), r.sqlDelete, id)
	if err != nil {
		log.Printf("Error al eliminar el item: %v", err)
		return err
	}

	return nil
}

func CreateNewElement[T any]() *T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	v := reflect.New(t).Elem()
	return v.Addr().Interface().(*T)
}

func ProcessTagSql(item interface{}) {
	itemValue := reflect.ValueOf(item).Elem()
	itemType := itemValue.Type()

	for i := 0; i < itemType.NumField(); i++ {
		func(i int) {
			field := itemType.Field(i)
			tag := field.Tag.Get("sql")
			if tag == "" {
				return
			}
			conn, err := config.DB.Conn(context.Background())
			if err != nil {
				log.Printf("Error al obtener la conexion: %v", err)
				return
			}
			defer conn.Close()
			fmt.Println(tag, itemValue.FieldByName("ID").Interface())

			tagParam := field.Tag.Get("sql_param")
			arrayParam := []interface{}{}
			if tagParam != "" {
				arrayTagParam := strings.Split(tagParam, ",")
				for _, param := range arrayTagParam {
					arrayParam = append(arrayParam, itemValue.FieldByName(param).Interface())
				}
			} else {
				arrayParam = append(arrayParam, itemValue.FieldByName("ID").Interface())
			}

			rows, err := conn.QueryContext(context.Background(), tag, arrayParam...)
			if err != nil {
				log.Printf("Error al ejecutar la consulta: %v", err)
				return
			}
			defer rows.Close()

			// Obtiene el tipo del campo y crea una nueva instancia
			fieldType := field.Type
			if fieldType.Kind() == reflect.Slice {
				sliceType := fieldType.Elem()
				sliceVal := reflect.MakeSlice(fieldType, 0, 0)

				// Itera sobre los resultados de la consulta
				for rows.Next() {
					newElem, _ := Scan2(sliceType, rows)
					sliceVal = reflect.Append(sliceVal, newElem)
				}

				// Establece el valor del campo en la estructura
				itemValue.FieldByName(field.Name).Set(sliceVal)
				return
			}
			if fieldType.Kind() == reflect.Ptr {
				if rows.Next() {
					ptrType := fieldType.Elem()
					elemVal, err := Scan2(ptrType, rows)
					if err != nil {
						log.Printf("Error al escanear la fila: %v", err)
						return
					}
					ptrVal := elemVal.Addr()
					fmt.Println(ptrVal)
					itemValue.FieldByName(field.Name).Set(ptrVal)
				}
				return
			}

			if fieldType.Kind() == reflect.Struct {
				if rows.Next() {
					elemVal, err := Scan2(fieldType, rows)
					if err != nil {
						log.Printf("Error al escanear la fila: %v", err)
						return
					}
					itemValue.FieldByName(field.Name).Set(elemVal)
				}
				return
			}
		}(i)
	}
}

type iRow interface {
	Scan(dest ...any) error
}

func Scan2(itemType reflect.Type, row iRow) (reflect.Value, error) {
	item := reflect.New(itemType).Elem()
	l := itemType.NumField()
	values := make([]interface{}, 0)

	for i := 0; i < l; i++ {
		field := itemType.Field(i)

		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}
		//fmt.Println(tag, field.Name, item.FieldByName(field.Name).Type())
		values = append(values, item.FieldByName(field.Name).Addr().Interface())
	}

	err := row.Scan(values...)
	if err != nil {
		log.Printf("Error al escanear la fila: %v", err)
	}

	ProcessTagSql(item.Addr().Interface())
	return item, err
}