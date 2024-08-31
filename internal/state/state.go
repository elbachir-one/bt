package state

import (
	t "github.com/LeperGnome/bt/internal/tree"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

type Operation int

const (
	Noop Operation = iota
	Move
	Copy
	Delete
	Go
	Insert
	InsertFile
	InsertDir
	Rename
)

func (o Operation) Repr() string {
	return []string{
		"",
		"moving",
		"copying",
		"confirm removing (y/n) of",
		"g",
		"create new (f)ile/(d)irectory",
		"enter new file name:",
		"enter new directory name:",
		"renaming",
	}[o]
}
func (o Operation) IsInput() bool {
	switch o {
	case InsertDir, InsertFile, Rename:
		return true
	default:
		return false
	}
}

type State struct {
	Tree     *t.Tree
	OpBuf    Operation
	InputBuf []rune
	watcher  *fsnotify.Watcher
}

func InitState(root string) (*State, <-chan NodeChange, error) {
	tree, err := t.InitTree(root, nil)
	if err != nil {
		return nil, nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	eventsChan := runFSWatcher(watcher)
	err = watcher.Add(tree.Root.Path)
	if err != nil {
		return nil, nil, err
	}
	return &State{
		Tree:     &tree,
		OpBuf:    Noop,
		InputBuf: []rune{},
		watcher:  watcher,
	}, eventsChan, nil
}

func (s *State) ProcessNodeChange(nodeChange NodeChange) tea.Cmd {
	err := s.Tree.RefreshNodeByPath(nodeChange.Path)
	if err != nil {
		panic(err) // TODO
	}
	return nil
}

func (s *State) ProcessKey(msg tea.KeyMsg) tea.Cmd {
	switch s.OpBuf {
	case Noop:
		return s.processKeyDefault(msg)
	case Move:
		return s.processKeyMove(msg)
	case Delete:
		return s.processKeyDelete(msg)
	case Copy:
		return s.processKeyCopy(msg)
	case Go:
		return s.processKeyGo(msg)
	case Insert:
		return s.processKeyInsert(msg)
	case InsertFile:
		return s.processKeyInsertFile(msg)
	case InsertDir:
		return s.processKeyInsertDir(msg)
	case Rename:
		return s.processKeyRename(msg)
	default:
		return s.processKeyDefault(msg)
	}
}
func (s *State) processKeyRename(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.RenameMarked(string(s.InputBuf))
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyInsert(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "f":
		s.OpBuf = InsertFile
	case "d":
		s.OpBuf = InsertDir
	default:
		s.OpBuf = Noop
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyInsertFile(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.CreateFileInCurrent(string(s.InputBuf))
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyInsertDir(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.CreateDirectoryInCurrent(string(s.InputBuf))
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyAnyInput(msg tea.KeyMsg) tea.Cmd {
	// TODO: better input handling? cursor?
	switch msg.String() {
	case "ctrl+c", "esc":
		s.OpBuf = Noop
		s.InputBuf = []rune{}
		s.Tree.DropMark()
	case "backspace":
		if l := len(s.InputBuf); l > 0 {
			s.InputBuf = s.InputBuf[:l-1]
		}
	default:
		s.InputBuf = append(s.InputBuf, msg.Runes...)
	}
	return nil
}
func (s *State) processKeyGo(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "g":
		s.OpBuf = Noop
		s.Tree.CurrentDir.SelectFirst()
	default:
		s.OpBuf = Noop
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyDelete(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y":
		err := s.Tree.DeleteMarked()
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
	default:
		s.OpBuf = Noop
		s.Tree.DropMark()
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyMove(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "p":
		err := s.Tree.MoveMarkedToCurrentDir()
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
	default:
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyCopy(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "p":
		err := s.Tree.CopyMarkedToCurrentDir()
		if err != nil {
			panic(err) // TODO
		}
		s.OpBuf = Noop
	default:
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyDefault(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		s.Tree.DropMark()
		s.OpBuf = Noop
	case "ctrl+c", "q":
		return tea.Quit
	case "j", "down":
		s.Tree.SelectNextChild()
	case "k", "up":
		s.Tree.SelectPreviousChild()
	case "l", "right":
		selected := s.Tree.GetSelectedChild()
		if selected != nil {
			s.watcher.Add(selected.Path)
		}
		err := s.Tree.SetSelectedChildAsCurrent()
		if err != nil {
			panic(err) // TODO
		}
	case "h", "left":
		s.Tree.SetParentAsCurrent()
	case "y":
		if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Copy
		}
	case "d":
		if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Move
		}
	case "D":
		if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Delete
		}
	case "g":
		s.OpBuf = Go
	case "G":
		s.Tree.CurrentDir.SelectLast()
	case "i":
		s.Tree.DropMark()
		s.OpBuf = Insert
	case "r":
		if ok := s.Tree.MarkSelectedChild(); ok {
			s.InputBuf = []rune(s.Tree.Marked.Info.Name())
			s.OpBuf = Rename
		}
	case "enter":
		err := s.Tree.CollapseOrExpandSelected()
		if err != nil {
			panic(err) // TODO
		}
	}
	return nil
}
