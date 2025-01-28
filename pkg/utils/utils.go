package utils

import (
	"personal-site/internal/config"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func Map[T any, U any](input []T, fn func(T) U) []U {
	result := make([]U, len(input))
	for i, v := range input {
		result[i] = fn(v)
	}
	return result
}

func GenerateToken(user_id int) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin":   true,
		"user_id": user_id,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	s, err := t.SignedString(config.SignKey)
	if err != nil {
		return "", err
	}
	return s, nil
}

func FormatTitle(filename string) string {
	return filename[:len(filename)-3]
}

func TitleToSlug(title string) string {
	titleLower := strings.ToLower(title)
	// remove all punctuation
	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	cleansedTitle := reg.ReplaceAllString(titleLower, "")
	slugArray := strings.Split(cleansedTitle, " ")
	slug := strings.Join(slugArray, "-")
	return slug
}

func ParseTags(contents string) string {
	tagsSection := strings.Split(contents, "---")[1]
	tagsSection = strings.ReplaceAll(tagsSection, "\n", "")
	tagsSection = strings.TrimPrefix(tagsSection, "tags:")
	tagsParts := strings.Split(tagsSection, "- ")[1:]
	tagsParts = Map(tagsParts, func(part string) string {
		return strings.Trim(part, " ")
	})

	return strings.Join(tagsParts, " ")
}
func CleanPostContent(content *string) {
	contentParts := strings.Split(*content, "---")
	contentParts = contentParts[2:]
	*content = strings.Join(contentParts, "")
}
