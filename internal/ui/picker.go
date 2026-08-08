package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yuedongze/roestat/internal/roest"
)

// machineItem adapts a roest.Machine to the bubbles/list default delegate.
type machineItem struct{ m roest.Machine }

func (i machineItem) Title() string       { return i.m.Name }
func (i machineItem) Description() string { return fmt.Sprintf("id %d · %s", i.m.ID, i.m.MachineID) }
func (i machineItem) FilterValue() string { return i.m.Name }

// pickerModel is the machine picker shown before entering the live view.
type pickerModel struct {
	list    list.Model
	loading bool
	err     error
}

func newPicker() pickerModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a machine to watch live"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	return pickerModel{list: l, loading: true}
}

func (p *pickerModel) setMachines(ms []roest.Machine) {
	items := make([]list.Item, len(ms))
	for i, m := range ms {
		items[i] = machineItem{m}
	}
	p.list.SetItems(items)
	p.loading = false
}

func (p *pickerModel) setSize(w, h int) { p.list.SetSize(w, h) }

func (p *pickerModel) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return cmd
}

func (p pickerModel) filtering() bool {
	return p.list.FilterState() == list.Filtering
}

func (p pickerModel) selected() (roest.Machine, bool) {
	it, ok := p.list.SelectedItem().(machineItem)
	return it.m, ok
}

func (p pickerModel) view() string {
	if p.err != nil {
		return errorStyle.Render("Failed to load machines: " + p.err.Error())
	}
	if p.loading {
		return spinnerHintStyle.Render("Loading machines…")
	}
	return p.list.View()
}
