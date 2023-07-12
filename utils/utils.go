package utils

import (
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

// GenerateRandomString genera una cadena aleatoria de longitud dada.
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[random.Intn(len(charset))]
	}

	return string(b)
}

// GetCurrentTimestamp devuelve la marca de tiempo actual en formato Unix.
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// StringToFile guarda una cadena en un archivo con la ruta especificada.
func StringToFile(content string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// FileToString lee el contenido de un archivo en la ruta especificada y lo devuelve como una cadena.
func FileToString(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	fileSize := stat.Size()
	content := make([]byte, fileSize)

	_, err = file.Read(content)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
