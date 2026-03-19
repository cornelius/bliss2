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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	styleListHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	stylePos         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMuted       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleActive      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	styleSectionHead = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
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

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("getting home directory: %w", err)
			}
			// Resolve symlinks to properly compare paths (handles macOS /var -> /private/var)
			realCwd, err := filepath.EvalSymlinks(cwd)
			if err != nil {
				realCwd = cwd
			}
			realHome, err := filepath.EvalSymlinks(home)
			if err != nil {
				realHome = home
			}
			if realCwd == realHome {
				return fmt.Errorf("cannot init a context in the home directory — use personal mode instead")
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
			if err := s.WriteContextMeta(contextUUID, name, cwd); err != nil {
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

			contextUUID, _, _ := blisscontext.FindContext(cwd)

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
	var all bool

	cmd := &cobra.Command{
		Use:   "list [list-name]",
		Short: "Show todos with position numbers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, _ := blisscontext.FindContext(cwd)

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			// --all: show every context plus personal, no session.
			if all {
				return listAll(s)
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

			first := true
			printList := func(header string, l list.List, resolve func(string) (todo.Todo, error)) {
				if !first {
					fmt.Println()
				}
				first = false
				fmt.Println(styleListHeader.Render("  " + header))
				empty := true
				for si, section := range l.Sections {
					if si > 0 {
						if section.Name != "" {
							fmt.Println(styleSectionHead.Render("      ── " + section.Name))
						} else {
							fmt.Println(styleSectionHead.Render("      ──"))
						}
					}
					for _, uuid := range section.Items {
						t, err := resolve(uuid)
						if err != nil {
							continue
						}
						fmt.Printf("  %s  %s\n", stylePos.Render(fmt.Sprintf("%2d", pos)), t.Title)
						session[pos] = uuid
						pos++
						empty = false
					}
				}
				if empty {
					fmt.Println(styleMuted.Render("    (no todos)"))
				}
			}

			inboxTodos, err := getInboxTodos(s, contextUUID)
			if err != nil {
				return err
			}

			inboxList := func() list.List {
				uuids := make([]string, len(inboxTodos))
				for i, t := range inboxTodos {
					uuids[i] = t.UUID
				}
				return list.List{Sections: []list.Section{{Items: uuids}}}
			}

			if filterList == "inbox" {
				printList("inbox", inboxList(), readTodo)
				return s.WriteSession(session)
			}

			if filterList != "" {
				l, err := s.ReadList(contextUUID, filterList)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", filterList, err)
				}
				printList(filterList, l, readTodo)
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
				if len(list.AllUUIDs(l)) == 0 {
					continue
				}
				printList(name, l, readTodo)
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
				if len(list.AllUUIDs(l)) == 0 {
					continue
				}
				printList(name, l, readTodoAny)
			}
			if len(inboxTodos) > 0 {
				printList("inbox", inboxList(), readTodo)
			}

			if pos == 1 {
				fmt.Println("(no todos)")
			}

			return s.WriteSession(session)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show todos from all contexts")
	return cmd
}

// listAll shows todos from all contexts plus personal todos.
func listAll(s *store.Store) error {
	contextUUIDs, err := s.ListContextUUIDs()
	if err != nil {
		return err
	}

	first := true
	printHeader := func(header string) {
		if !first {
			fmt.Println()
		}
		first = false
		fmt.Println(styleListHeader.Render("  " + header))
	}

	for _, uuid := range contextUUIDs {
		name, _, _ := s.ReadContextMeta(uuid)
		todos, err := s.ListTodos(uuid)
		if err != nil || len(todos) == 0 {
			continue
		}
		printHeader(name)
		for _, t := range todos {
			fmt.Printf("    %s\n", t.Title)
		}
	}

	// Personal todos.
	personalTodos, err := s.ListTodos("")
	if err == nil && len(personalTodos) > 0 {
		printHeader("personal")
		for _, t := range personalTodos {
			fmt.Printf("    %s\n", t.Title)
		}
	}

	return nil
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

			contextUUID, _, _ := blisscontext.FindContext(cwd)

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

			contextUUID, _, _ := blisscontext.FindContext(cwd)

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

			contextUUID, _, _ := blisscontext.FindContext(cwd)

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

			m := ui.NewCheckModel(s, contextUUID, items, filterList)
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

			contextUUID, _, _ := blisscontext.FindContext(cwd)

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

	if filterList != "" {
		l, err := s.ReadList(contextUUID, filterList)
		if err != nil {
			return nil, fmt.Errorf("reading list %q: %w", filterList, err)
		}
		resolve := func(uuid string) (todo.Todo, error) { return s.ReadTodo(contextUUID, uuid) }
		return appendSectionItems(items, l, filterList, contextUUID, resolve), nil
	}

	// No filter: context lists, then personal lists, then inbox.
	listNames, err := s.ListNames(contextUUID)
	if err != nil {
		return nil, err
	}
	for _, name := range listNames {
		l, err := s.ReadList(contextUUID, name)
		if err != nil {
			continue
		}
		if len(list.AllUUIDs(l)) == 0 {
			continue
		}
		resolve := func(uuid string) (todo.Todo, error) { return s.ReadTodo(contextUUID, uuid) }
		items = append(items, ui.CheckItem{SectionHeader: name, IsListHeader: true})
		items = appendSectionItems(items, l, name, contextUUID, resolve)
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
		if len(list.AllUUIDs(l)) == 0 {
			continue
		}
		resolve := func(uuid string) (todo.Todo, error) { return s.FindTodo(uuid) }
		items = append(items, ui.CheckItem{SectionHeader: name, IsListHeader: true})
		items = appendSectionItems(items, l, name, "", resolve)
	}

	inboxTodos, err := getInboxTodos(s, contextUUID)
	if err != nil {
		return nil, err
	}
	if len(inboxTodos) > 0 {
		items = append(items, ui.CheckItem{SectionHeader: "inbox", IsListHeader: true})
		for i := range inboxTodos {
			t := inboxTodos[i]
			items = append(items, ui.CheckItem{Todo: &t})
		}
	}

	return items, nil
}

// appendSectionItems appends CheckItems for each section of a list, including section separators.
// listName and listCtx are set on todo items so section insertion works from any view.
func appendSectionItems(items []ui.CheckItem, l list.List, listName, listCtx string, resolve func(string) (todo.Todo, error)) []ui.CheckItem {
	for si, section := range l.Sections {
		if si > 0 {
			items = append(items, ui.CheckItem{
				IsSectionHeader: true,
				SectionHeader:   section.Name,
				SectionName:     section.Name,
				SectionIdx:      si,
			})
		}
		for _, uuid := range section.Items {
			t, err := resolve(uuid)
			if err != nil {
				continue
			}
			tc := t
			items = append(items, ui.CheckItem{Todo: &tc, ListName: listName, ListContextUUID: listCtx})
		}
	}
	return items
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
				name, _, err := s.ReadContextMeta(uuid)
				if err != nil {
					name = uuid
				}

				todos, err := s.ListTodos(uuid)
				if err != nil {
					todos = nil
				}

				count := styleMuted.Render(fmt.Sprintf("%d", len(todos)))
				if uuid == activeUUID {
					fmt.Printf("  %s  %s\n", styleActive.Render("* "+name), count)
				} else {
					fmt.Printf("    %-28s%s\n", name, count)
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
				contextUUID, _, _ = blisscontext.FindContext(cwd)
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
				fmt.Printf("  %s  %s\n", styleMuted.Render(e.Time.Format("Jan 02 15:04")), msg)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show history across all contexts")
	return cmd
}
