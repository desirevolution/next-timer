package client

import (
	"github.com/desirevolution/next-timer/internal/protocol"
	"strings"
	"testing"
)

func TestFavoriteBySelectorDoesNotGuessDuplicateNames(t *testing.T) {
	favorites := []protocol.Template{{ID: "f-1", Name: "Pause", Group: "Work"}, {ID: "f-2", Name: "pause", Group: "Home"}}
	if _, err := favoriteBySelector(favorites, "PAUSE"); err == nil || !strings.Contains(err.Error(), "mehrdeutig") {
		t.Fatalf("missing ambiguity error: %v", err)
	}
	t2, err := favoriteBySelector(favorites, "#2")
	if err != nil || t2.ID != "f-2" {
		t.Fatalf("index: %v %v", t2, err)
	}
	t1, err := favoriteBySelector(favorites, "f-1")
	if err != nil || t1.ID != "f-1" {
		t.Fatalf("ID: %v %v", t1, err)
	}
}
