package handlers

import (
	"net/http"
	"reflect"

	"github.com/arturoeanton/go-struct2serve/repositories"
	"github.com/arturoeanton/go-struct2serve/services"
	"github.com/labstack/echo/v4"
)

type IHandler[T any] interface {
	Name() string
	GetAll(c echo.Context) error
	GetByID(c echo.Context) error
	Create(c echo.Context) error
	DeleteByID(c echo.Context) error
	Update(c echo.Context) error
}

type Handler[T any] struct {
	service services.IService[T]
	name    string
}

func NewHandler[T any]() *Handler[T] {

	return &Handler[T]{
		name: "items",
		service: services.NewService[T](
			repositories.NewRepository[T](),
		),
	}
}

func (h *Handler[T]) Name() string {
	return h.name
}

func (h *Handler[T]) GetAll(c echo.Context) error {
	items, err := h.service.GetAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	return c.JSON(http.StatusOK, items)
}

func (h *Handler[T]) GetByID(c echo.Context) error {
	id := c.Param("id")
	item, err := h.service.GetByID(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	return c.JSON(http.StatusOK, item)
}

func (h *Handler[T]) Create(c echo.Context) error {
	item := new(T)
	if err := c.Bind(item); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	id, err := h.service.Create(item)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	reflect.ValueOf(item).Elem().FieldByName("ID").SetInt(id)
	return c.JSON(http.StatusOK, id)
}

func (h *Handler[T]) DeleteByID(c echo.Context) error {
	id := c.Param("id")
	err := h.service.Delete(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	return c.JSON(http.StatusOK, id)
}

func (h *Handler[T]) Update(c echo.Context) error {
	item := new(T)
	if err := c.Bind(item); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	err := h.service.Update(item)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get " + h.Name(),
		})
	}
	return c.JSON(http.StatusNoContent, nil)
}
