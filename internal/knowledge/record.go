package knowledge

type Record struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	Description string `json:"description"`
}

func NewRecord(name, description string) *Record {
	return &Record{
		Name:        name,
		Description: description,
		Size:        0,
	}
}

func (k *Record) Grade() string {
	switch {
	case k.Size < 200:
		return "pretty small"
	case k.Size < 300:
		return "small"
	case k.Size < 500:
		return "moderate"
	case k.Size < 600:
		return "large"
	default:
		return "too large!"
	}
}
