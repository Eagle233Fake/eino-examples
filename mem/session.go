package mem

import (
	"encoding/json"
	"os"
	"slices"
	"time"

	"github.com/cloudwego/eino/schema"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	msgs      []*schema.AgenticMessage
	filePath  string
}

func (s *Session) Append(msg *schema.AgenticMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	s.msgs = append(s.msgs, msg)
	return nil
}

func (s *Session) GetMsgs() []*schema.AgenticMessage {
	return slices.Clone(s.msgs)
}
