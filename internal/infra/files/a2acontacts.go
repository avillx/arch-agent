package files

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

const a2aContactsFile = "a2a_contacts.json"

type A2AFiles struct {
	fs       *FileSystem
	contacts map[agent.ID][]*service.A2AContact
	mu       sync.RWMutex
}

func NewA2AFiles(fs *FileSystem) (*A2AFiles, error) {

	data, err := fs.ReadFile(a2aContactsFile)
	if err != nil {
		return nil, err
	}

	var contacts map[agent.ID][]*service.A2AContact
	if err := json.Unmarshal(data, &contacts); err != nil {
		return nil, err
	}

	return &A2AFiles{
		fs:       fs,
		contacts: contacts,
	}, nil
}

func (f *A2AFiles) Get(agentID agent.ID) ([]*service.A2AContact, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	contacts, ok := f.contacts[agentID]

	if !ok {
		return nil, fmt.Errorf("agent %s has no contacts to call other agents", agentID)
	}

	return contacts, nil
}

func (f *A2AFiles) Save(agentID agent.ID, newContact *service.A2AContact) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	f.mu.Lock()
	agentContacts, ok := f.contacts[agentID]
	if !ok {
		agentContacts = []*service.A2AContact{}
	}
	agentContacts = append(agentContacts, newContact)
	f.mu.Unlock()

	return f.flush()
}

func (f *A2AFiles) Delete(agentID agent.ID, contactAgentID agent.ID) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	agentContacts, ok := f.contacts[agentID]
	if !ok {
		return fmt.Errorf("agent %s aleready have no any contacts", agentID)
	}
	agentContacts = slices.DeleteFunc(agentContacts, func(contact *service.A2AContact) bool {
		return contact.ID == contactAgentID
	})

	return f.flush()
}

func (f *A2AFiles) flush() error {
	data, err := json.MarshalIndent(f.contacts, "", "	")
	if err != nil {
		return err
	}

	return f.fs.WriteToFile(a2aContactsFile, data)
}
