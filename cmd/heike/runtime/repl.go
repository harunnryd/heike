package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/store"
)

type REPL struct {
	components *RuntimeComponents
	reader     *bufio.Reader
	sessionID  string
}

func NewREPL(components *RuntimeComponents) *REPL {
	sessionID := fmt.Sprintf("cli-%d", time.Now().Unix())

	components.StoreWorker.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "CLI Session",
		Status:    "active",
		Metadata:  map[string]string{"source": "cli"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	return &REPL{
		components: components,
		reader:     bufio.NewReader(os.Stdin),
		sessionID:  sessionID,
	}
}

func (r *REPL) Start() error {
	fmt.Printf("Heike Interactive Session: %s\n", r.sessionID)
	fmt.Println("Type '/exit' to quit.")

	for {
		select {
		case <-r.components.Ctx.Done():
			return nil
		default:
			if err := r.readLine(); err != nil {
				if err == io.EOF || err.Error() == "received shutdown signal" {
					return nil
				}
				continue
			}
		}
	}
}

func (r *REPL) readLine() error {
	fmt.Print("> ")
	text, err := r.reader.ReadString('\n')
	if err != nil {
		return err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if text == "/exit" {
		return io.EOF
	}

	evt := ingress.NewEvent("cli", ingress.TypeUserMessage, r.sessionID, text, map[string]string{
		"source": "cli",
	})

	return r.components.Ingress.Submit(r.components.Ctx, &evt)
}
