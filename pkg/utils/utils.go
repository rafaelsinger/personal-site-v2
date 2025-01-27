package utils

import (
	"personal-site/internal/config"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/html"
)

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

// thanks chat
// ParseTags extracts <li> elements only within the <ul> that follows <p>tags:</p> and is between <hr> elements.
func ParseTags(n *html.Node) []string {
	var tags []string
	var inTagsSection bool // Track whether we're inside the desired section

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Check for <hr> to mark start/end of tags section
			if n.Data == "hr" {
				inTagsSection = !inTagsSection // Toggle state
			}

			// Detect <p> with the exact text "tags:"
			if n.Data == "p" && inTagsSection && n.FirstChild != nil && strings.TrimSpace(n.FirstChild.Data) == "tags:" {
				// Check if the next sibling is a <ul>
				for sibling := n.NextSibling; sibling != nil; sibling = sibling.NextSibling {
					if sibling.Type == html.ElementNode && sibling.Data == "ul" {
						// Process <li> elements inside this <ul>
						for child := sibling.FirstChild; child != nil; child = child.NextSibling {
							if child.Type == html.ElementNode && child.Data == "li" && child.FirstChild != nil {
								tags = append(tags, strings.TrimSpace(child.FirstChild.Data))
							}
						}
						break // Only process the first <ul> found after <p>tags:</p>
					}
				}
			}
		}

		// Traverse child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(n)
	return tags
}
