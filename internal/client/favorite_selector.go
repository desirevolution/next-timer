package client

import (
	"errors"
	"github.com/desirevolution/next-timer/internal/protocol"
	"strconv"
	"strings"
)

// favoriteBySelector resolves IDs and #positions directly. A name is only
// accepted when it uniquely identifies one template across all groups.
func favoriteBySelector(templates []protocol.Template, selector string) (*protocol.Template, error) {
	if strings.HasPrefix(selector, "#") {
		index, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
		if err != nil || index < 1 || index > len(templates) {
			return nil, errors.New("ungültige Favoritenposition")
		}
		return &templates[index-1], nil
	}
	for i := range templates {
		if templates[i].ID == selector {
			return &templates[i], nil
		}
	}
	var match *protocol.Template
	for i := range templates {
		if strings.EqualFold(templates[i].Name, selector) {
			if match != nil {
				return nil, errors.New("Favoritenname ist mehrdeutig; bitte #Nummer oder ID verwenden (siehe nexttimer favorites)")
			}
			match = &templates[i]
		}
	}
	if match == nil {
		return nil, errors.New("Favorit nicht gefunden")
	}
	return match, nil
}
