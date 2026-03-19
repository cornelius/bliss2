package main

import (
	blisscontext "bliss/internal/context"
	"bliss/internal/list"
	"bliss/internal/store"
	"bliss/internal/todo"
	"bliss/internal/ui"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	styleListHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	stylePos        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMuted      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleActive     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "bliss",
		Short: "bliss — a personal todo management tool",
	}

	root.AddCommand(
		initCmd(),
		addCmd(),
		listCmd(),
		doneCmd(),
		moveCmd(),
		checkCmd(),
		groomCmd(),
		contextsCmd(),
		historyCmd(),
	)

	return root
}

// initCmd implements `bliss init [--name <name>]`
func initCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a bliss context in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			// Check for parent context
			if parentUUID, parentDir, err := blisscontext.FindContext(filepath.Dir(cwd)); err == nil {
				fmt.Printf("Note: parent context found in %s (UUID: %s)\n", parentDir, parentUUID)
			}

			// Generate UUID for new context
			contextUUID := uuid.New().String()

			// Derive name from cwd if not provided
			if name == "" {
				name = filepath.Base(cwd)
			}

			// Initialize store
			s, err := store.Init()
			if err != nil {
				return fmt.Errorf("initializing store: %w", err)
			}

			// Write context metadata
			if err := s.WriteContextMeta(contextUUID, name); err != nil {
				return fmt.Errorf("writing context meta: %w", err)
			}

			// Write .bliss-context marker
			if err := blisscontext.WriteContextFile(cwd, contextUUID); err != nil {
				return fmt.Errorf("writing .bliss-context: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("init context %s (%s)", name, contextUUID)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Initialized bliss context: %s (%s)\n", name, contextUUID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Context name (default: current directory name)")
	return cmd
}

// addCmd implements `bliss add <title> [--list <name>] [--urgent]`
func addCmd() *cobra.Command {
	var listName string
	var urgent bool

	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new todo",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if urgent && listName == "" {
				return fmt.Errorf("--urgent requires --list")
			}

			var title string
			if len(args) > 0 {
				title = strings.Join(args, " ")
			} else {
				fi, err := os.Stdin.Stat()
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				if fi.Mode()&os.ModeCharDevice != 0 {
					fmt.Print("Title: ")
				}
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil && err != io.EOF {
					return fmt.Errorf("reading title: %w", err)
				}
				title = strings.TrimSpace(line)
				if title == "" {
					return fmt.Errorf("title cannot be empty")
				}
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			t := todo.Todo{
				UUID:  uuid.New().String(),
				Title: title,
			}

			if err := s.WriteTodo(contextUUID, t); err != nil {
				return fmt.Errorf("writing todo: %w", err)
			}

			if listName != "" {
				l, err := s.ReadList(contextUUID, listName)
				if err != nil {
					return fmt.Errorf("reading list: %w", err)
				}
				list.Add(&l, t.UUID, urgent)
				if err := s.WriteList(contextUUID, listName, l); err != nil {
					return fmt.Errorf("writing list: %w", err)
				}
			}

			commitMsg := fmt.Sprintf("add todo %q", title)
			if err := s.Commit(commitMsg); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			if listName != "" {
				fmt.Printf("Added to [%s]: %s\n", listName, title)
			} else {
				fmt.Printf("Added: %s\n", title)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&listName, "list", "l", "", "Add to a named list")
	cmd.Flags().BoolVar(&urgent, "urgent", false, "Prepend to list (requires --list)")
	return cmd
}

// listCmd implements `bliss list [list-name]`
func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [list-name]",
		Short: "Show todos with position numbers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			var filterList string
			if len(args) > 0 {
				filterList = args[0]
			}

			session := make(map[int]string)
			pos := 1

			readTodo := func(uuid string) (todo.Todo, error) {
				return s.ReadTodo(contextUUID, uuid)
			}
			readTodoAny := func(uuid string) (todo.Todo, error) {
				return s.FindTodo(uuid)
			}

			printList := func(header string, uuids []string, resolve func(string) (todo.Todo, error)) {
				fmt.Println(styleListHeader.Render("[" + header + "]"))
				if len(uuids) == 0 {
					fmt.Println(styleMuted.Render("  (no todos)"))
					return
				}
				for _, uuid := range uuids {
					t, err := resolve(uuid)
					if err != nil {
						continue
					}
					fmt.Printf("  %s %s\n", stylePos.Render(fmt.Sprintf("%d.", pos)), t.Title)
					session[pos] = uuid
					pos++
				}
			}

			inboxTodos, err := getInboxTodos(s, contextUUID)
			if err != nil {
				return err
			}
			inboxUUIDs := make([]string, len(inboxTodos))
			for i, t := range inboxTodos {
				inboxUUIDs[i] = t.UUID
			}

			if filterList == "inbox" {
				printList("inbox", inboxUUIDs, readTodo)
				return s.WriteSession(session)
			}

			if filterList != "" {
				// Check context list first, then personal list.
				l, err := s.ReadList(contextUUID, filterList)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", filterList, err)
				}
				printList(filterList, list.AllUUIDs(l), readTodo)
				return s.WriteSession(session)
			}

			// No filter: context lists, then personal lists, then inbox.
			listNames, err := s.ListNames(contextUUID)
			if err != nil {
				return err
			}
			for _, name := range listNames {
				l, err := s.ReadList(contextUUID, name)
				if err != nil {
					continue
				}
				uuids := list.AllUUIDs(l)
				if len(uuids) == 0 {
					continue
				}
				printList(name, uuids, readTodo)
			}
			personalNames, err := s.PersonalListNames()
			if err != nil {
				return err
			}
			for _, name := range personalNames {
				l, err := s.ReadList("", name)
				if err != nil {
					continue
				}
				uuids := list.AllUUIDs(l)
				if len(uuids) == 0 {
					continue
				}
				printList(name, uuids, readTodoAny)
			}
			if len(inboxTodos) > 0 {
				printList("inbox", inboxUUIDs, readTodo)
			}

			if pos == 1 {
				fmt.Println("(no todos)")
			}

			return s.WriteSession(session)
		},
	}
}

// doneCmd implements `bliss done <number>`
func doneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <number|uuid>",
		Short: "Complete a todo by position number or UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			todoUUID, err := resolveTodo(args[0], s, contextUUID)
			if err != nil {
				return err
			}

			// Read title for confirmation message
			t, err := s.ReadTodo(contextUUID, todoUUID)
			if err != nil {
				return fmt.Errorf("reading todo: %w", err)
			}

			if err := s.DeleteTodo(contextUUID, todoUUID); err != nil {
				return fmt.Errorf("deleting todo: %w", err)
			}

			if err := s.RemoveFromAllLists(contextUUID, todoUUID); err != nil {
				return fmt.Errorf("removing from lists: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("complete %q", t.Title)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Done: %s\n", t.Title)
			return nil
		},
	}
}

func moveCmd() *cobra.Command {
	var listName string
	var urgent bool

	cmd := &cobra.Command{
		Use:   "move <number|uuid> --list <name>",
		Short: "Move a todo to a list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if listName == "" {
				return fmt.Errorf("--list is required")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return err
			}

			todoUUID, err := resolveTodo(args[0], s, contextUUID)
			if err != nil {
				return err
			}

			t, err := s.ReadTodo(contextUUID, todoUUID)
			if err != nil {
				return fmt.Errorf("reading todo: %w", err)
			}

			if err := s.RemoveFromAllLists(contextUUID, todoUUID); err != nil {
				return fmt.Errorf("removing from lists: %w", err)
			}

			l, err := s.ReadList(contextUUID, listName)
			if err != nil {
				return fmt.Errorf("reading list %q: %w", listName, err)
			}
			list.Add(&l, todoUUID, urgent)
			if err := s.WriteList(contextUUID, listName, l); err != nil {
				return fmt.Errorf("writing list: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("move %q to %s", t.Title, listName)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Moved to [%s]: %s\n", listName, t.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&listName, "list", "l", "", "Target list")
	cmd.Flags().BoolVar(&urgent, "urgent", false, "Place at top of list")
	return cmd
}

// checkCmd implements `bliss check [list-name]`
func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [list-name]",
		Short: "Interactive todo viewer",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			var filterList string
			if len(args) > 0 {
				filterList = args[0]
			}

			items, err := buildCheckItems(s, contextUUID, filterList)
			if err != nil {
				return err
			}

			m := ui.NewCheckModel(s, contextUUID, items)
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running check UI: %w", err)
			}
			return nil
		},
	}
}

// groomCmd implements `bliss groom [list-name]`
func groomCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "groom [list-name]",
		Short: "Interactive grooming mode",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, err := blisscontext.FindContext(cwd)
			if err != nil {
				return err
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			startList := "inbox"
			if len(args) > 0 {
				startList = args[0]
			}

			m := ui.NewGroomModel(s, contextUUID, ui.DefaultKanbanOrder, startList)
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running groom UI: %w", err)
			}
			return nil
		},
	}
}

// buildCheckItems builds the list of CheckItems for the check command.
func buildCheckItems(s *store.Store, contextUUID, filterList string) ([]ui.CheckItem, error) {
	var items []ui.CheckItem

	if filterList == "inbox" || filterList == "" {
		if filterList == "inbox" {
			todos, err := getInboxTodos(s, contextUUID)
			if err != nil {
				return nil, err
			}
			for i := range todos {
				t := todos[i]
				items = append(items, ui.CheckItem{Todo: &t})
			}
			return items, nil
		}

		// Context lists, then personal lists, then inbox.
		listNames, err := s.ListNames(contextUUID)
		if err != nil {
			return nil, err
		}
		for _, name := range listNames {
			l, err := s.ReadList(contextUUID, name)
			if err != nil {
				continue
			}
			uuids := list.AllUUIDs(l)
			if len(uuids) == 0 {
				continue
			}
			items = append(items, ui.CheckItem{SectionHeader: name})
			for _, uuid := range uuids {
				t, err := s.ReadTodo(contextUUID, uuid)
				if err != nil {
					continue
				}
				tc := t
				items = append(items, ui.CheckItem{Todo: &tc})
			}
		}
		personalNames, err := s.PersonalListNames()
		if err != nil {
			return nil, err
		}
		for _, name := range personalNames {
			l, err := s.ReadList("", name)
			if err != nil {
				continue
			}
			uuids := list.AllUUIDs(l)
			if len(uuids) == 0 {
				continue
			}
			items = append(items, ui.CheckItem{SectionHeader: name})
			for _, uuid := range uuids {
				t, err := s.FindTodo(uuid)
				if err != nil {
					continue
				}
				tc := t
				items = append(items, ui.CheckItem{Todo: &tc})
			}
		}

		inboxTodos, err := getInboxTodos(s, contextUUID)
		if err != nil {
			return nil, err
		}
		if len(inboxTodos) > 0 {
			items = append(items, ui.CheckItem{SectionHeader: "inbox"})
			for i := range inboxTodos {
				t := inboxTodos[i]
				items = append(items, ui.CheckItem{Todo: &t})
			}
		}
	} else {
		l, err := s.ReadList(contextUUID, filterList)
		if err != nil {
			return nil, fmt.Errorf("reading list %q: %w", filterList, err)
		}
		for _, uuid := range list.AllUUIDs(l) {
			t, err := s.ReadTodo(contextUUID, uuid)
			if err != nil {
				continue
			}
			tc := t
			items = append(items, ui.CheckItem{Todo: &tc})
		}
	}

	return items, nil
}

// resolveTodo resolves a position number or UUID string to a todo UUID.
func resolveTodo(arg string, s *store.Store, contextUUID string) (string, error) {
	if _, err := uuid.Parse(arg); err == nil {
		return arg, nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil {
		return "", fmt.Errorf("expected a position number or UUID, got %q", arg)
	}
	session, err := s.ReadSession()
	if err != nil {
		return "", err
	}
	todoUUID, ok := session[n]
	if !ok {
		return "", fmt.Errorf("no todo at position %d (run 'bliss list' to refresh)", n)
	}
	return todoUUID, nil
}

// getInboxTodos returns todos that are not in any named list.
func getInboxTodos(s *store.Store, contextUUID string) ([]todo.Todo, error) {
	todos, err := s.ListTodos(contextUUID)
	if err != nil {
		return nil, err
	}

	listNames, err := s.ListNames(contextUUID)
	if err != nil {
		return nil, err
	}

	listedUUIDs := make(map[string]bool)
	for _, name := range listNames {
		l, err := s.ReadList(contextUUID, name)
		if err != nil {
			continue
		}
		for _, uuid := range list.AllUUIDs(l) {
			listedUUIDs[uuid] = true
		}
	}

	var inbox []todo.Todo
	for _, t := range todos {
		if !listedUUIDs[t.UUID] {
			inbox = append(inbox, t)
		}
	}
	return inbox, nil
}

func contextsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contexts",
		Short: "List all contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.Open()
			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			activeUUID, _, _ := blisscontext.FindContext(cwd)

			uuids, err := s.ListContextUUIDs()
			if err != nil {
				return err
			}

			for _, uuid := range uuids {
				name, err := s.ReadContextMeta(uuid)
				if err != nil {
					name = uuid
				}

				todos, err := s.ListTodos(uuid)
				if err != nil {
					todos = nil
				}

				count := styleMuted.Render(fmt.Sprintf("%d todos", len(todos)))
				if uuid == activeUUID {
					fmt.Printf("%s %s\n", styleActive.Render("* "+name), count)
				} else {
					fmt.Printf("  %-30s %s\n", name, count)
				}
			}
			return nil
		},
	}
}

func historyCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show history of changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.Open()
			if err != nil {
				return err
			}

			contextUUID := ""
			if !all {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}
				contextUUID, _, err = blisscontext.FindContext(cwd)
				if err != nil {
					return err
				}
			}

			entries, err := s.ReadHistory(contextUUID)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("(no history)")
				return nil
			}

			for _, e := range entries {
				msg := strings.TrimPrefix(strings.TrimSpace(e.Message), "bliss: ")
				fmt.Printf("%s  %s\n", styleMuted.Render(e.Time.Format(time.DateTime)), msg)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show history across all contexts")
	return cmd
}
