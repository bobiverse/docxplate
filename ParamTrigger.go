package docxplate

import (
	"bytes"
	"fmt"
	"log"
	"strings"
)

// On - trigger events when command to aply
const (
	TriggerOnUnknown string = ":unknown"
	TriggerOnEmpty   string = ":empty"
	TriggerOnValue   string = ":="
)

// Command - what to do when triggered
const (
	TriggerCommandRemove = ":remove"
	TriggerCommandClear  = ":clear"
)

// Scope - scope of affected elements by command
const (
	TriggerScopePlaceholder = ":placeholder"
	TriggerScopeCell        = ":cell"
	TriggerScopeRow         = ":row"
	TriggerScopeList        = ":list"
	TriggerScopeTable       = ":table"
	TriggerScopeSection     = ":section" // table, list..
)

// ParamTrigger - param trigger command
// {{Key :On:Command:Scope}}
// {{MyParam :empty:remove:list}} -- Read as: "`remove` `list` on `empty` value"
type ParamTrigger struct {
	raw string

	On      string
	Command string
	Scope   string
}

// NewParamTrigger - take raw ":empty:remove:list" and make trigger and its fields from it
func NewParamTrigger(raw []byte) *ParamTrigger {
	raw = bytes.TrimSpace(raw)
	raw = bytes.ToLower(raw)

	// init with defaults
	tr := &ParamTrigger{
		raw:     string(raw),
		On:      TriggerOnUnknown,
		Command: TriggerCommandRemove,
		Scope:   TriggerScopeRow,
	}

	// Always must start with ":"
	if !strings.HasPrefix(tr.raw, ":") {
		return nil
	}

	// Remove the first ":" so split parts counting is more readable
	// Split into parts
	parts := strings.Split(tr.raw[1:], ":")

	var countCommandParts = 0
	for _, part := range parts {
		switch part {
		case "unknown", "empty", "=":
			countCommandParts++
			tr.On = ":" + part
		case "remove", "clear":
			countCommandParts++
			tr.Command = ":" + part
		case "placeholder", "cell", "row", "list", "table", "section":
			countCommandParts++
			tr.Scope = ":" + part
		}
	}

	if countCommandParts != 3 {
		return nil
	}

	if !tr.isValid() {
		return nil
	}

	return tr
}

// Validate trigger
func (tr *ParamTrigger) isValid() bool {

	// On
	if !inSlice(tr.On, []string{
		TriggerOnUnknown,
		TriggerOnEmpty,
		TriggerOnValue,
	}) {
		log.Printf("ERROR: No such trigger on [%s]", tr.On)
		return false
	}

	// Command
	if !inSlice(tr.Command, []string{
		TriggerCommandClear,
		TriggerCommandRemove,
	}) {
		log.Printf("ERROR: No such trigger command [%s]", tr.Command)
		return false
	}

	// Scope
	if !inSlice(tr.Scope, []string{
		TriggerScopePlaceholder,
		TriggerScopeCell,
		TriggerScopeRow,
		TriggerScopeList,
		TriggerScopeTable,
		TriggerScopeSection,
	}) {
		log.Printf("ERROR: No such trigger scope [%s]", tr.Scope)
		return false
	}

	return true
}

// String - return rebuilt trigger string
func (tr *ParamTrigger) String() string {
	if tr == nil {
		return ""
	}
	s := fmt.Sprintf("%s%s%s", tr.On, tr.Command, tr.Scope)
	return s
}
