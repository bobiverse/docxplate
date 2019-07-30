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

// NewParamTrigger - take raw ":empty:remove:list" and make
// trigger and it's fields from it
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
	// remove first ":" so split parts counting is more readable
	tr.raw = strings.TrimPrefix(tr.raw, ":")

	// Must be at least set two parts ":empty:remove"
	if strings.Count(tr.raw, ":") < 2 {
		return nil
	}
	// Extract fields
	arr := strings.SplitN(tr.raw, ":", 3)
	tr.On = ":" + arr[0]
	tr.Command = ":" + arr[1]

	// Scope: set default if not found
	if len(arr) >= 3 {
		tr.Scope = ":" + arr[2]
	}

	if !tr.isValid() {
		return nil
	}

	// fmt.Printf("TRIGGER: %+v", tr)
	return tr
}

// Validate trigger
func (tr *ParamTrigger) isValid() bool {

	// On
	if !inSlice(tr.On, []string{
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
