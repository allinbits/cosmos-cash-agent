package ui

import "github.com/allinbits/cosmos-cash-agent/pkg/model"

type State struct {
	Messages        map[string][]model.TextMessage `json:"contacts"`
	SelectedContact int                            `json:"selected_contact"`
	Credentials     []string                       `json:"credentials"`
	Address         string                         `json:"address"`
}

func NewState() *State {
	return &State{
		Messages:        make(map[string][]model.TextMessage, 0),
		Credentials:     make([]string, 0),
		SelectedContact: -1,
	}
}
