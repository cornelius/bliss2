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
	"sort"
	"time"
)

var (
	styleListHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	stylePos         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleMuted       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleActive      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	styleSectionHead = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// Status command styles.
// Color is used sparingly: one brand accent (cyan on "bliss"), grey for
// de-emphasis, and green/amber/red only for meaningful status signals.
var (
	stTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4")) // brand
	stMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))             // labels, secondary info
	stPath   = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))             // paths (darker)
	stBold   = lipgloss.NewStyle().Bold(true)                                        // active names, counts
	stGood   = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))             // synced / ok
	stWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))             // ahead
	stBad    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))             // behind / error
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "bliss",
		Short:        "bliss — a personal todo management tool",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(
		initCmd(),
		addCmd(),
		listCmd(),
		doneCmd(),
		moveCmd(),
		checkCmd(),
		groomCmd(),
		statusCmd(),
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
	var personal bool

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

			// listCtxUUID is "" for personal mode (--personal or outside a context).
			listCtxUUID := contextUUID
			if personal || contextUUID == "" {
				listCtxUUID = ""
			}

			session := make(map[int]string)
			pos := 1

			readTodo := func(uuid string) (todo.Todo, error) {
				return s.ReadTodo(listCtxUUID, uuid)
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

			inboxTodos, err := getInboxTodos(s, listCtxUUID)
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
				l, err := s.ReadList(listCtxUUID, filterList)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", filterList, err)
				}
				printList(filterList, l, readTodo)
				return s.WriteSession(session)
			}

			// No filter: show context lists or personal lists, never both.
			listNames, err := s.ListNames(listCtxUUID)
			if err != nil {
				return err
			}
			for _, name := range listNames {
				l, err := s.ReadList(listCtxUUID, name)
				if err != nil {
					continue
				}
				if len(list.AllUUIDs(l)) == 0 {
					continue
				}
				printList(name, l, readTodo)
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
	cmd.Flags().BoolVarP(&personal, "personal", "p", false, "Show personal lists")
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

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show context status and git sync",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			activeUUID, _, _ := blisscontext.FindContext(cwd)
			personalMode := activeUUID == ""

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			// ── header ───────────────────────────────────────────────────
			date := stMuted.Render(time.Now().Format("Mon Jan 02, 2006"))
			fmt.Println(stTitle.Render("bliss status") + "  " + date)
			fmt.Println()

			// ── contexts ─────────────────────────────────────────────────
			contextUUIDs, err := s.ListContextUUIDs()
			if err != nil {
				return err
			}
			type ctxRow struct {
				uuid   string
				name   string
				path   string
				active bool
			}
			var rows []ctxRow
			for _, uuid := range contextUUIDs {
				name, path, _ := s.ReadContextMeta(uuid)
				rows = append(rows, ctxRow{uuid, name, path, uuid == activeUUID})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
			for _, r := range rows {
				counts := statusListCounts(s, r.uuid)
				if len(counts) == 0 {
					continue
				}
				fmt.Println(renderContextRow(r.active, r.name, r.path, counts))
			}

			// ── personal ─────────────────────────────────────────────────
			personalCounts := statusListCounts(s, "")
			if len(personalCounts) > 0 {
				fmt.Println(renderPersonalRow(personalMode, personalCounts))
			}

			// ── store / git ───────────────────────────────────────────────
			fmt.Println()
			remote, ahead, behind, _ := s.GitSyncStatus()
			storeLabel := stMuted.Render(fmt.Sprintf("%-8s", "Store:"))
			remoteLabel := stMuted.Render(fmt.Sprintf("%-8s", "Remote:"))
			syncLabel := stMuted.Render(fmt.Sprintf("%-8s", "Sync:"))
			fmt.Println(storeLabel + s.Path())
			if remote != "" {
				fmt.Println(remoteLabel + remote)
			}
			syncVal := renderSyncStatus(remote, ahead, behind)
			fmt.Println(syncLabel + syncVal)

			return nil
		},
	}
}

type listCount struct {
	name  string
	count int
}

// statusListCounts returns per-list todo counts for a context (or personal mode if "").
// Lists with zero todos are omitted.
func statusListCounts(s *store.Store, contextUUID string) []listCount {
	var names []string
	if contextUUID == "" {
		names, _ = s.PersonalListNames()
	} else {
		names, _ = s.ListNames(contextUUID)
	}

	var counts []listCount
	for _, name := range names {
		l, err := s.ReadList(contextUUID, name)
		if err != nil {
			continue
		}
		n := len(list.AllUUIDs(l))
		if n > 0 {
			counts = append(counts, listCount{name, n})
		}
	}

	inboxCount := statusInboxCount(s, contextUUID)
	if inboxCount > 0 {
		counts = append(counts, listCount{"inbox", inboxCount})
	}

	return counts
}

func statusInboxCount(s *store.Store, contextUUID string) int {
	todos, err := getInboxTodos(s, contextUUID)
	if err != nil {
		return 0
	}
	return len(todos)
}

// isContextPathFresh checks whether a path still contains a .bliss-context pointing to uuid.
func isContextPathFresh(path, uuid string) bool {
	data, err := os.ReadFile(filepath.Join(path, ".bliss-context"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == uuid
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

// listLabelStyle returns a muted grey style for list name labels.
// List names are structural labels, not data — they should not draw attention.
func listLabelStyle(_ string) lipgloss.Style {
	return stMuted
}

// sortedCounts returns counts in a semantic display order: today → this-week →
// next-week → later → custom lists → bugs → inbox.
func sortedCounts(counts []listCount) []listCount {
	priority := func(name string) int {
		switch name {
		case "today":
			return 0
		case "this-week":
			return 1
		case "next-week":
			return 2
		case "later":
			return 3
		case "bugs":
			return 5
		case "inbox":
			return 9
		default:
			return 4 // custom lists: after later, before bugs
		}
	}
	out := make([]listCount, len(counts))
	copy(out, counts)
	sort.Slice(out, func(i, j int) bool {
		pi, pj := priority(out[i].name), priority(out[j].name)
		if pi != pj {
			return pi < pj
		}
		return out[i].name < out[j].name
	})
	return out
}

// shortenHomePath replaces the home directory prefix in path with "~".
func shortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// renderCounts formats list counts as "name: count  name: count  ...".
func renderCounts(counts []listCount) string {
	counts = sortedCounts(counts)
	var parts []string
	for _, lc := range counts {
		label := listLabelStyle(lc.name).Render(lc.name + ":")
		count := stBold.Render(strconv.Itoa(lc.count))
		parts = append(parts, label+" "+count)
	}
	return strings.Join(parts, "  ")
}

// renderContextRow renders one context line in the status output.
//
// Column layout: [12 label+indicator][10 name][20 path][2 sep][list data]
func renderContextRow(active bool, name, path string, counts []listCount) string {
	// Label + indicator column (12 chars visual)
	var prefix string
	if active {
		prefix = stMuted.Render("Context:") + "  " + stTitle.Render(">") + " "
	} else {
		prefix = stMuted.Render("Context:") + "    "
	}

	// Name column (10 chars): bold for active, muted for inactive
	nameStr := fmt.Sprintf("%-10s", name)
	var nameStyled string
	if active {
		nameStyled = stBold.Render(nameStr)
	} else {
		nameStyled = stMuted.Render(nameStr)
	}

	// Path column (20 chars, home-shortened, truncated if needed) + 2-char separator.
	// Total path+separator = 22 chars, matching the personal row blank span.
	const pathWidth = 20
	short := shortenHomePath(path)
	if len(short) > pathWidth {
		short = short[:pathWidth-1] + "…"
	}
	pathStyled := stPath.Render(fmt.Sprintf("%-*s", pathWidth, short))

	return prefix + nameStyled + pathStyled + "  " + renderCounts(counts)
}

// renderPersonalRow renders the personal todos line aligned with context rows.
func renderPersonalRow(active bool, counts []listCount) string {
	// Label + indicator column (12 chars visual)
	var prefix string
	if active {
		prefix = stMuted.Render("Personal:") + " " + stTitle.Render(">") + " "
	} else {
		prefix = stMuted.Render("Personal:") + "   "
	}
	// Name + path columns are blank (10 + 20 + 2 = 32 spaces)
	return prefix + strings.Repeat(" ", 32) + renderCounts(counts)
}

// renderSyncStatus formats the sync state value for the Sync: line.
// Green/amber/red are used here because sync state is a meaningful status signal.
func renderSyncStatus(remote string, ahead, behind int) string {
	if remote == "" {
		return stMuted.Render("no remote")
	}
	if ahead == 0 && behind == 0 {
		return stGood.Render("✓ synced")
	}
	var parts []string
	if ahead > 0 {
		parts = append(parts, stWarn.Render(fmt.Sprintf("↑%d ahead", ahead)))
	}
	if behind > 0 {
		parts = append(parts, stBad.Render(fmt.Sprintf("↓%d behind", behind)))
	}
	return strings.Join(parts, "  ")
}
