package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

type formField struct {
	key   string
	label string
	input textinput.Model
}

type formModel struct {
	title  string
	fields []formField
	index  int
}

func newForm(title string, labels, keys []string) formModel {
	fields := make([]formField, 0, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Focus()
		ti.Prompt = "> "
		ti.PromptStyle = styles.Subtle
		ti.TextStyle = styles.Item
		ti.Cursor.Style = styles.Accent
		if i != 0 {
			ti.Blur()
		}
		fields = append(fields, formField{key: keys[i], label: label, input: ti})
	}
	return formModel{title: title, fields: fields}
}

func newFormWithValues(title string, labels, keys []string, values map[string]string) formModel {
	form := newForm(title, labels, keys)
	form.setValues(values)
	return form
}

func (f *formModel) next() {
	if f.index < len(f.fields)-1 {
		f.fields[f.index].input.Blur()
		f.index++
		f.fields[f.index].input.Focus()
	}
}

func (f *formModel) prev() {
	if f.index > 0 {
		f.fields[f.index].input.Blur()
		f.index--
		f.fields[f.index].input.Focus()
	}
}

func (f *formModel) values() map[string]string {
	res := map[string]string{}
	for i := range f.fields {
		field := &f.fields[i]
		res[field.key] = strings.TrimSpace(field.input.Value())
	}
	return res
}

func (f *formModel) setValues(values map[string]string) {
	for i := range f.fields {
		if val, ok := values[f.fields[i].key]; ok {
			f.fields[i].input.SetValue(val)
		}
	}
}
