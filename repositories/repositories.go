package repositories

import (
	"context"
	"database/sql"
	"log"
	"reflect"
	"strings"

	"github.com/arturoeanton/go-struct2serve/config"
	"github.com/arturoeanton/go-struct2serve/utils"
)

var (
	S2S              string = "s2s"
	S2S_ID           string = "s2s_id"
	S2S_TABLE_NAME   string = "s2s_table_name"
	S2S_UPDATE_VALUE string = "s2s_update_value"
	S2S_PARAM        string = "s2s_param"
)

type IRepository[T any] interface {
	GetAll() ([]*T, error)
	GetByID(id interface{}) (*T, error)
	GetByCriteria(criteria string, args ...interface{}) ([]*T, error)
	Create(item *T) (int64, error)
	Update(item *T) error
	Delete(id interface{}) error

	GetTableName() string
	GetTags() []string
	GetTagsName() map[string]string

	SetDepth(depth int) IRepository[T]
	GetDepth() int
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
	defaultDepth     int
}

func NewRepository[T any]() *Repository[T] {
	item := CreateNewElement[T]()
	itemType := reflect.TypeOf(*item)
	return NewRepositoryWithTable[T](utils.ToSnakeCase(itemType.Name()))
}

func NewRepositoryWithTable[T any](table string) *Repository[T] {
	item := CreateNewElement[T]()

	r := &Repository[T]{
		table:        table,
		defaultDepth: 2,
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
	itemType = reflect.TypeOf(*item)
	r.sqlAll = createSelectSection(itemType) + createFromSection(itemType)
	r.sqlGetByCriteria = createSelectSection(itemType) + createFromSection(itemType) + " WHERE "
	r.sqlGetByID = createSelectSection(itemType) + createFromSection(itemType) + " WHERE id = ?"
	r.sqlCreate = sqlCreate
	r.sqlUpdate = sqlUpdate
	r.sqlDelete = "DELETE FROM " + table + " WHERE id = ?"

	return r
}

func createFromSection(itemType reflect.Type) string {
	tableName := utils.ToSnakeCase(itemType.Name())
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		tag := field.Tag.Get(S2S_TABLE_NAME)
		if tag != "" {
			tableName = tag
			break
		}
	}

	return " FROM " + tableName + "  "
}

func createSelectSection(itemType reflect.Type) string {

	fieldList := ""
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" {
			continue
		}
		if fieldList != "" {
			fieldList += ", "
		}
		fieldList += tag
	}
	return "SELECT " + fieldList + " "
}

func (r *Repository[T]) GetAll() ([]*T, error) {
	conn, err := config.DB.Conn(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), r.sqlAll)
	if err != nil {
		log.Printf("Error al ejecutar la consulta[009]: %v", err)
		return nil, err
	}
	defer rows.Close()
	items := []*T{}
	for rows.Next() {
		item := CreateNewElement[T]()
		v, err := Scan2(reflect.TypeOf(*item), rows, r.defaultDepth)
		//err := Scan[T](item, rows)
		if err != nil {
			if config.FlagLog {
				log.Printf("Error al escanear la fila[008]: %v", err)
			}
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
		if config.FlagLog {
			log.Printf("Error al ejecutar la consulta[007]: %v", err)
		}
		return nil, err
	}
	defer rows.Close()
	items := []*T{}
	for rows.Next() {
		item := CreateNewElement[T]()
		v, err := Scan2(reflect.TypeOf(*item), rows, r.defaultDepth)
		//err := Scan[T](item, rows)
		if err != nil {
			if config.FlagLog {
				log.Printf("Error al escanear la fila[006]: %v", err)
			}
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
	v, err := Scan2(reflect.TypeOf(*item), row, r.defaultDepth)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No se encontró el usuario
		}
		if config.FlagLog {
			log.Printf("Error al escanear la fila[005]: %v", err)
		}
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
		if config.FlagLog {
			log.Printf("Error al crear el item: %v", err)
		}
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
	itemValue := reflect.ValueOf(item).Elem()
	itemType := itemValue.Type()
	fieldIdName := "ID"
	for i := 0; i < itemType.NumField(); i++ {
		tagID := itemType.Field(i).Tag.Get(S2S_ID)
		if tagID == "true" {
			fieldIdName = itemType.Field(i).Name
			break
		}
	}
	for _, tag := range r.tags {
		value := reflect.ValueOf(*item).FieldByName(r.tagName[tag])
		field, b := reflect.TypeOf(*item).FieldByName(r.tagName[tag])
		if b {
			tagSqlUpdateValue := field.Tag.Get(S2S_UPDATE_VALUE)
			if tagSqlUpdateValue != "" {
				tagSqlUpdateValueArray := strings.Split(tagSqlUpdateValue, ".")
				if len(tagSqlUpdateValueArray) == 2 {
					v := reflect.ValueOf(*item).FieldByName(tagSqlUpdateValueArray[0])
					if v.Kind() == reflect.Ptr {
						v = v.Elem()
					}

					value = v.FieldByName(tagSqlUpdateValueArray[1])
				}
			}
		}

		fieldsValues = append(fieldsValues, value.Interface())
	}
	fieldsValues = append(fieldsValues, reflect.ValueOf(*item).FieldByName(fieldIdName).Interface())

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

func (r *Repository[T]) GetTableName() string {
	return r.table
}

func (r *Repository[T]) GetTags() []string {
	return r.tags
}

func (r *Repository[T]) GetTagsName() map[string]string {
	// clone map
	m := make(map[string]string)
	for k, v := range r.tagName {
		m[k] = v
	}
	return m
}

func (r *Repository[T]) SetDepth(depth int) IRepository[T] {
	r.defaultDepth = depth
	return r
}
func (r *Repository[T]) GetDepth() int {
	return r.defaultDepth
}

func CreateNewElement[T any]() *T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	v := reflect.New(t).Elem()
	return v.Addr().Interface().(*T)
}

func ProcessTagSql(item interface{}, depth int) {
	itemValue := reflect.ValueOf(item).Elem()
	itemType := itemValue.Type()

	fieldIdName := "ID"
	for i := 0; i < itemType.NumField(); i++ {
		tagID := itemType.Field(i).Tag.Get(S2S_ID)
		if tagID == "true" {
			fieldIdName = itemType.Field(i).Name
			break
		}
	}
	for i := 0; i < itemType.NumField(); i++ {
		func(i int) {
			field := itemType.Field(i)
			tag := field.Tag.Get("s2s")
			if tag == "" {
				return
			}

			conn, err := config.DB.Conn(context.Background())
			if err != nil {
				log.Printf("Error al obtener la conexion: %v", err)
				return
			}
			defer conn.Close()
			if config.FlagLog {
				log.Println(tag, itemValue.FieldByName(fieldIdName).Interface())
			}
			tagParam := field.Tag.Get(S2S_PARAM)
			arrayParam := []interface{}{}
			if tagParam != "" {
				arrayTagParam := strings.Split(tagParam, ",")
				for _, param := range arrayTagParam {
					arrayParam = append(arrayParam, itemValue.FieldByName(param).Interface())
				}
			} else {
				arrayParam = append(arrayParam, itemValue.FieldByName(fieldIdName).Interface())
			}

			fieldType := field.Type
			lowTag := strings.ToLower(tag)
			if !strings.HasPrefix(lowTag, "select") {
				var subItemType reflect.Type
				if fieldType.Kind() == reflect.Ptr {
					ptrType := fieldType.Elem()
					if ptrType.Kind() == reflect.Struct {
						subItemType = reflect.New(ptrType).Elem().Type()
					} else if ptrType.Kind() == reflect.Slice {
						subItemType = ptrType.Elem()
					}
				} else {
					if fieldType.Kind() == reflect.Struct {
						subItemType = fieldType
					} else if fieldType.Kind() == reflect.Slice {
						subItemType = fieldType.Elem()
					}
				}

				if !strings.HasPrefix(lowTag, "from") {
					if !strings.HasPrefix(lowTag, "where") {
						if !strings.ContainsAny(lowTag, " =><?-!") {
							tag = tag + " = ? "
						}
						tag = " WHERE " + tag
					}

					tag = createFromSection(subItemType) + tag
				}

				//fmt.Println("55>>", subItemType)
				tag = createSelectSection(subItemType) + tag
			}

			rows, err := conn.QueryContext(context.Background(), tag, arrayParam...)
			if err != nil {
				log.Printf("Error al ejecutar la consulta[004]: %v", err)
				return
			}
			defer rows.Close()

			// Obtiene el tipo del campo y crea una nueva instancia

			if fieldType.Kind() == reflect.Slice {
				sliceType := fieldType.Elem()
				sliceVal := reflect.MakeSlice(fieldType, 0, 0)

				// Itera sobre los resultados de la consulta
				for rows.Next() {
					newElem, _ := Scan2(sliceType, rows, depth)
					sliceVal = reflect.Append(sliceVal, newElem)
				}

				// Establece el valor del campo en la estructura
				itemValue.FieldByName(field.Name).Set(sliceVal)
				return
			}
			if fieldType.Kind() == reflect.Ptr {
				ptrType := fieldType.Elem()
				if ptrType.Kind() == reflect.Struct {
					if rows.Next() {
						elemVal, err := Scan2(ptrType, rows, depth)
						if err != nil {
							if config.FlagLog {
								log.Printf("Error al escanear la fila[003]: %v", err)
							}
							return
						}
						ptrVal := elemVal.Addr()
						itemValue.FieldByName(field.Name).Set(ptrVal)
						return
					}
				}
				if ptrType.Kind() == reflect.Slice {
					sliceType := ptrType.Elem()
					sliceVal := reflect.MakeSlice(ptrType, 0, 0)

					// Itera sobre los resultados de la consulta
					for rows.Next() {
						newElem, _ := Scan2(sliceType, rows, depth)
						sliceVal = reflect.Append(sliceVal, newElem)
					}
					ptr := reflect.New(sliceVal.Type())
					ptr.Elem().Set(sliceVal)
					// Establece el valor del campo en la estructura
					itemValue.FieldByName(field.Name).Set(ptr)
					return
				}
			}

			if fieldType.Kind() == reflect.Struct {
				if rows.Next() {
					elemVal, err := Scan2(fieldType, rows, depth)
					if err != nil {
						if config.FlagLog {
							log.Printf("Error al escanear la fila[002]: %v", err)
						}
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

func Scan2(itemType reflect.Type, row iRow, depth int) (reflect.Value, error) {
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
		if config.FlagLog {
			log.Printf("Error al escanear la fila[001]: %v", err)
		}
	}
	depth = depth - 1
	if depth > 0 {
		ProcessTagSql(item.Addr().Interface(), depth)
	}
	return item, err
}
