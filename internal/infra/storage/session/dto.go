package sessionfiles

import (
	"arch-agent/internal/app/session"
	"arch-agent/internal/infra/storage"
	"encoding/json"
)

type SessionDTO struct {
	ID       string               `json:"id"`
	Tokens   int                  `json:"tokens"`
	Messages []storage.MessageDTO `json:"messages"`
}

func unmarshalSession(data []byte) (*session.Session, error) {
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return dtoToSession(dto)
}

func dtoToSession(dto SessionDTO) (*session.Session, error) {
	msgs, err := storage.DtoToMessages(dto.Messages)
	if err != nil {
		return nil, err
	}
	return session.NewRestoredSession(dto.Tokens, msgs), nil
}

func marshalSession(s *session.Session) ([]byte, error) {
	return json.Marshal(
		SessionDTO{
			Tokens:   s.Tokens,
			Messages: storage.MessagesToDTO(s.Messages()),
		},
	)
}
