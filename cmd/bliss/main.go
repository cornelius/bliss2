package main

import (
	blisscontext "bliss/internal/context"
	"bliss/internal/list"
	"bliss/internal/slug"
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
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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
		Short:        "∴ bliss — a personal todo management tool",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(
		initCmd(),
		addCmd(),
		showCmd(),
		listCmd(),
		doneCmd(),
		moveCmd(),
		checkCmd(),
		groomCmd(),
		statusCmd(),
		syncCmd(),
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

			// Refuse to re-initialize a directory that already has a context.
			if existing, err := blisscontext.ReadContextFile(cwd); err == nil && existing != "" {
				return fmt.Errorf("already initialized as context %s", existing)
			}

			// Check for parent context
			if parentName, parentDir, err := blisscontext.FindContext(filepath.Dir(cwd)); err == nil {
				fmt.Printf("Note: parent context found in %s (%s)\n", parentDir, parentName)
			}

			// Derive name from cwd if not provided, then slugify
			if name == "" {
				name = filepath.Base(cwd)
			}
			contextName := slug.Slugify(name)

			// Initialize store
			s, err := store.Init()
			if err != nil {
				return fmt.Errorf("initializing store: %w", err)
			}

			// If the context already exists, link this directory to it (cross-machine story).
			if s.ContextExists(contextName) {
				fmt.Printf("Context '%s' already exists. Link this directory to it? [Y/n] ", contextName)
				var answer string
				fmt.Scanln(&answer)
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "" && answer != "y" && answer != "yes" {
					return fmt.Errorf("aborted")
				}

				if err := s.WriteContextMeta(contextName, cwd); err != nil {
					return fmt.Errorf("linking context: %w", err)
				}
				if err := blisscontext.WriteContextFile(cwd, contextName); err != nil {
					return fmt.Errorf("writing .bliss-context: %w", err)
				}
				if err := s.Commit(fmt.Sprintf("link context %s to %s", contextName, cwd)); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				short := shortenHomePath(cwd)
				fmt.Println(stMuted.Render("Linked to existing context") + "  " +
					stMuted.Render("Context:") + " " + stBold.Render(contextName) +
					"  " + stMuted.Render("Path:") + " " + stPath.Render(short))
				return nil
			}

			// Write context metadata
			if err := s.WriteContextMeta(contextName, cwd); err != nil {
				return fmt.Errorf("writing context meta: %w", err)
			}

			// Write .bliss-context marker
			if err := blisscontext.WriteContextFile(cwd, contextName); err != nil {
				return fmt.Errorf("writing .bliss-context: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("init context %s", contextName)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			short := shortenHomePath(cwd)
			fmt.Println(stMuted.Render("Initialized") + "  " +
				stMuted.Render("Context:") + " " + stBold.Render(contextName) +
				"  " + stMuted.Render("Path:") + " " + stPath.Render(short))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Context name (default: current directory name)")
	return cmd
}

// resolveContextName determines the active context name for a command.
// Priority: explicit --context flag > .bliss-context in CWD tree > personal ("").
// If a context name is found but the context does not exist in the store, the
// user is offered to sync first; if it still does not exist, an error is returned.
func resolveContextName(s *store.Store, flagValue, cwd string) (string, error) {
	var contextName string
	if flagValue != "" {
		contextName = flagValue
	} else {
		name, _, err := blisscontext.FindContext(cwd)
		if err != nil {
			// No .bliss-context found — personal mode.
			return "", nil
		}
		contextName = name
	}

	if !s.ContextExists(contextName) {
		fmt.Printf("Context '%s' not found locally.\n", contextName)
		fmt.Printf("Run 'bliss sync' to fetch it from remote? [Y/n] ")
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			if _, _, err := s.Sync(); err != nil {
				fmt.Fprintf(os.Stderr, "sync failed: %v\n", err)
			}
		}
		if !s.ContextExists(contextName) {
			return "", fmt.Errorf("context '%s' not found. Has it been initialized on another machine with 'bliss init'?", contextName)
		}
	}

	return contextName, nil
}

// addCmd implements `bliss add <title> [--list <name>] [--urgent] [--context <name>]`
func addCmd() *cobra.Command {
	var listName string
	var urgent bool
	var contextFlag string

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

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			contextName, err := resolveContextName(s, contextFlag, cwd)
			if err != nil {
				return err
			}

			t := todo.Todo{
				UUID:  uuid.New().String(),
				Title: title,
			}

			if err := s.WriteTodo(contextName, t); err != nil {
				return fmt.Errorf("writing todo: %w", err)
			}

			if listName != "" {
				l, err := s.ReadList(contextName, listName)
				if err != nil {
					return fmt.Errorf("reading list: %w", err)
				}
				list.Add(&l, t.UUID, urgent)
				if err := s.WriteList(contextName, listName, l); err != nil {
					return fmt.Errorf("writing list: %w", err)
				}
			}

			commitMsg := fmt.Sprintf("add todo %q", title)
			if err := s.Commit(commitMsg); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			if listName != "" {
				fmt.Println(stMuted.Render("Added to") + " " + stBold.Render(listName) + stMuted.Render(":") + " " + title)
			} else {
				fmt.Println(stMuted.Render("Added:") + " " + title)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&listName, "list", "l", "", "Add to a named list")
	cmd.Flags().BoolVar(&urgent, "urgent", false, "Prepend to list (requires --list)")
	cmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Use a specific context by name")
	return cmd
}

// listCmd implements `bliss list [list-name]`
// showCmd implements `bliss show [list-name]`
// Focus mode: same scoping as bliss list, but incoming is omitted unless non-empty.
// Future: will hide deferred todos and apply scene filtering.
func showCmd() *cobra.Command {
	var contextFlag string

	cmd := &cobra.Command{
		Use:   "show [list-name]",
		Short: "Show actionable todos (focus mode)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			contextName, err := resolveContextName(s, contextFlag, cwd)
			if err != nil {
				return err
			}
			contextUUID := contextName

			var filterList string
			if len(args) > 0 {
				filterList = args[0]
			}

			session := make(map[int]string)
			pos := 1

			resolve := func(uuid string) (todo.Todo, error) {
				return s.ReadTodo(contextUUID, uuid)
			}

			incomingTodos, err := getIncomingTodos(s, contextUUID)
			if err != nil {
				return err
			}
			incomingAsList := func() list.List {
				uuids := make([]string, len(incomingTodos))
				for i, t := range incomingTodos {
					uuids[i] = t.UUID
				}
				return list.List{Sections: []list.Section{{Items: uuids}}}
			}

			// Filtered view: no list name, no indent, numbers start at column 0.
			printFiltered := func(l list.List) {
				for si, section := range l.Sections {
					if si > 0 {
						fmt.Println(listSectionDelim(pos, section.Name))
					}
					for _, uuid := range section.Items {
						t, err := resolve(uuid)
						if err != nil {
							continue
						}
						fmt.Printf("%d  %s\n", pos, t.Title)
						session[pos] = uuid
						pos++
					}
				}
			}

			if filterList == "incoming" {
				printFiltered(incomingAsList())
				return s.WriteSession(session)
			}

			if filterList != "" {
				l, err := s.ReadList(contextUUID, filterList)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", filterList, err)
				}
				printFiltered(l)
				return s.WriteSession(session)
			}

			// Unfiltered view: list name bold, items indented with right-aligned number.
			first := true
			printOne := func(name string, l list.List) {
				if !first {
					fmt.Println()
				}
				first = false
				fmt.Println(stBold.Render(name))
				for si, section := range l.Sections {
					if si > 0 {
						fmt.Println(listSectionDelim(pos, section.Name))
					}
					for _, uuid := range section.Items {
						t, err := resolve(uuid)
						if err != nil {
							continue
						}
						fmt.Printf("%3d  %s\n", pos, t.Title)
						session[pos] = uuid
						pos++
					}
				}
			}

			listNames, err := s.ListNames(contextUUID)
			if err != nil {
				return err
			}
			for _, name := range sortListNames(listNames) {
				l, err := s.ReadList(contextUUID, name)
				if err != nil {
					continue
				}
				if len(list.AllUUIDs(l)) == 0 {
					continue
				}
				printOne(name, l)
			}
			if len(incomingTodos) > 0 {
				printOne("incoming", incomingAsList())
			}

			if pos == 1 {
				fmt.Println(stMuted.Render("All done. Nothing left to do."))
			}

			return s.WriteSession(session)
		},
	}

	cmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Use a specific context by name")
	return cmd
}

func listCmd() *cobra.Command {
	var all bool
	var personal bool
	var contextFlag string

	cmd := &cobra.Command{
		Use:   "list [list-name]",
		Short: "Show todos with position numbers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			if all {
				return listAll(s)
			}

			var filterList string
			if len(args) > 0 {
				filterList = args[0]
			}

			// Resolve context: --personal overrides everything to personal mode.
			var listCtxName string
			if !personal {
				resolved, err := resolveContextName(s, contextFlag, cwd)
				if err != nil {
					return err
				}
				listCtxName = resolved
			}

			// ── header ───────────────────────────────────────────────────
			fmt.Print(stTitle.Render("∴ bliss list") + "  ")
			if listCtxName == "" {
				fmt.Print(stMuted.Render("Personal"))
				if filterList != "" {
					fmt.Print("  " + stMuted.Render("List:") + " " + stBold.Render(filterList))
				}
			} else {
				ctxPath, _ := s.ReadContextMeta(listCtxName)
				fmt.Print(stMuted.Render("Context:") + " " + stBold.Render(listCtxName))
				if filterList != "" {
					fmt.Print("  " + stMuted.Render("List:") + " " + stBold.Render(filterList))
				}
				fmt.Print("  " + stMuted.Render("Path:") + " " + stPath.Render(shortenHomePath(ctxPath)))
			}
			fmt.Println()
			fmt.Println()

			// ── content ───────────────────────────────────────────────────
			session := make(map[int]string)
			pos := 1
			first := true

			resolve := func(uuid string) (todo.Todo, error) {
				return s.ReadTodo(listCtxName, uuid)
			}

			printOne := func(name string, l list.List, showName bool) {
				if !first {
					fmt.Println()
				}
				first = false
				if showName {
					fmt.Println(stBold.Render(name))
				}
				for si, section := range l.Sections {
					if si > 0 {
						fmt.Println(listSectionDelim(pos, section.Name))
					}
					for _, uuid := range section.Items {
						t, err := resolve(uuid)
						if err != nil {
							continue
						}
						fmt.Printf("%3d  %s\n", pos, t.Title)
						session[pos] = uuid
						pos++
					}
				}
			}

			incomingTodos, err := getIncomingTodos(s, listCtxName)
			if err != nil {
				return err
			}
			incomingAsList := func() list.List {
				uuids := make([]string, len(incomingTodos))
				for i, t := range incomingTodos {
					uuids[i] = t.UUID
				}
				return list.List{Sections: []list.Section{{Items: uuids}}}
			}

			if filterList == "incoming" {
				printOne("incoming", incomingAsList(), false)
				return s.WriteSession(session)
			}

			if filterList != "" {
				l, err := s.ReadList(listCtxName, filterList)
				if err != nil {
					return fmt.Errorf("reading list %q: %w", filterList, err)
				}
				printOne(filterList, l, false)
				return s.WriteSession(session)
			}

			// No filter: all lists in semantic order, then incoming.
			listNames, err := s.ListNames(listCtxName)
			if err != nil {
				return err
			}
			for _, name := range sortListNames(listNames) {
				l, err := s.ReadList(listCtxName, name)
				if err != nil {
					continue
				}
				if len(list.AllUUIDs(l)) == 0 {
					continue
				}
				printOne(name, l, true)
			}
			if len(incomingTodos) > 0 {
				printOne("incoming", incomingAsList(), true)
			}

			if pos == 1 {
				fmt.Println(stMuted.Render("(no todos)"))
			}

			return s.WriteSession(session)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show todos from all contexts")
	cmd.Flags().BoolVarP(&personal, "personal", "p", false, "Show personal lists")
	cmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Use a specific context by name")
	return cmd
}

// listAll shows todos from all contexts plus personal, grouped by list.
// No position numbers are assigned (spans multiple contexts).
func listAll(s *store.Store) error {
	// Header
	fmt.Println(stTitle.Render("∴ bliss list --all"))

	contextNames, err := s.ListContextNames()
	if err != nil {
		return err
	}

	type ctxInfo struct{ name, path string }
	var ctxs []ctxInfo
	for _, name := range contextNames {
		path, _ := s.ReadContextMeta(name)
		ctxs = append(ctxs, ctxInfo{name, path})
	}
	sort.Slice(ctxs, func(i, j int) bool { return ctxs[i].name < ctxs[j].name })

	// printAllList prints one named list (no position numbers). Returns true if anything printed.
	printAllList := func(ctxName, name string, l list.List, showName bool) bool {
		if len(list.AllUUIDs(l)) == 0 {
			return false
		}
		if showName {
			fmt.Println(stBold.Render(name))
		}
		for si, section := range l.Sections {
			if si > 0 {
				fmt.Println(listSectionDelim(1, section.Name))
			}
			for _, id := range section.Items {
				t, err := s.ReadTodo(ctxName, id)
				if err != nil {
					continue
				}
				fmt.Printf("  %s\n", t.Title)
			}
		}
		return true
	}

	for _, ctx := range ctxs {
		todos, err := s.ListTodos(ctx.name)
		if err != nil || len(todos) == 0 {
			continue
		}

		fmt.Println()
		fmt.Println(stMuted.Render("Context:") + " " + stBold.Render(ctx.name) +
			"  " + stMuted.Render("Path:") + " " + stPath.Render(shortenHomePath(ctx.path)))
		fmt.Println()

		firstList := true
		listNames, _ := s.ListNames(ctx.name)
		for _, name := range sortListNames(listNames) {
			l, err := s.ReadList(ctx.name, name)
			if err != nil {
				continue
			}
			if !firstList {
				fmt.Println()
			}
			if printAllList(ctx.name, name, l, true) {
				firstList = false
			}
		}

		incomingTodos, _ := getIncomingTodos(s, ctx.name)
		if len(incomingTodos) > 0 {
			if !firstList {
				fmt.Println()
			}
			uuids := make([]string, len(incomingTodos))
			for i, t := range incomingTodos {
				uuids[i] = t.UUID
			}
			printAllList(ctx.name, "incoming", list.List{Sections: []list.Section{{Items: uuids}}}, true)
		}
	}

	// Personal scope.
	personalTodos, err := s.ListTodos("")
	if err == nil && len(personalTodos) > 0 {
		fmt.Println()
		fmt.Println(stMuted.Render("Personal"))
		fmt.Println()

		firstList := true
		personalNames, _ := s.PersonalListNames()
		for _, name := range sortListNames(personalNames) {
			l, err := s.ReadList("", name)
			if err != nil {
				continue
			}
			if !firstList {
				fmt.Println()
			}
			if printAllList("", name, l, true) {
				firstList = false
			}
		}

		personalIncoming, _ := getIncomingTodos(s, "")
		if len(personalIncoming) > 0 {
			if !firstList {
				fmt.Println()
			}
			fmt.Println(stBold.Render("incoming"))
			for _, t := range personalIncoming {
				fmt.Printf("  %s\n", t.Title)
			}
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

			contextName, _, _ := blisscontext.FindContext(cwd)

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			todoUUID, err := resolveTodo(args[0], s, contextName)
			if err != nil {
				return err
			}

			// Read title for confirmation message
			t, err := s.ReadTodo(contextName, todoUUID)
			if err != nil {
				return fmt.Errorf("reading todo: %w", err)
			}

			if err := s.DeleteTodo(contextName, todoUUID); err != nil {
				return fmt.Errorf("deleting todo: %w", err)
			}

			if err := s.RemoveFromAllLists(contextName, todoUUID); err != nil {
				return fmt.Errorf("removing from lists: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("complete %q", t.Title)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Println(stMuted.Render("Done:") + " " + t.Title)
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

			contextName, _, _ := blisscontext.FindContext(cwd)

			s, err := store.Open()
			if err != nil {
				return err
			}

			todoUUID, err := resolveTodo(args[0], s, contextName)
			if err != nil {
				return err
			}

			t, err := s.ReadTodo(contextName, todoUUID)
			if err != nil {
				return fmt.Errorf("reading todo: %w", err)
			}

			if err := s.RemoveFromAllLists(contextName, todoUUID); err != nil {
				return fmt.Errorf("removing from lists: %w", err)
			}

			l, err := s.ReadList(contextName, listName)
			if err != nil {
				return fmt.Errorf("reading list %q: %w", listName, err)
			}
			list.Add(&l, todoUUID, urgent)
			if err := s.WriteList(contextName, listName, l); err != nil {
				return fmt.Errorf("writing list: %w", err)
			}

			if err := s.Commit(fmt.Sprintf("move %q to %s", t.Title, listName)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Println(stMuted.Render("Moved to") + " " + stBold.Render(listName) + stMuted.Render(":") + " " + t.Title)
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
			// Note: --context flag not yet supported for check; use CWD-based detection.

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

			contextName, _, _ := blisscontext.FindContext(cwd)
			// Note: --context flag not yet supported for groom; use CWD-based detection.

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			startList := "incoming"
			if len(args) > 0 {
				startList = args[0]
			}

			m := ui.NewGroomModel(s, contextName, ui.DefaultKanbanOrder, startList)
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

	if filterList == "incoming" {
		todos, err := getIncomingTodos(s, contextUUID)
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

	// No filter: context lists, then personal lists, then incoming.
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

	incomingTodos, err := getIncomingTodos(s, contextUUID)
	if err != nil {
		return nil, err
	}
	if len(incomingTodos) > 0 {
		items = append(items, ui.CheckItem{SectionHeader: "incoming", IsListHeader: true})
		for i := range incomingTodos {
			t := incomingTodos[i]
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

// getIncomingTodos returns todos that are not in any named list.
func getIncomingTodos(s *store.Store, contextUUID string) ([]todo.Todo, error) {
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

	var incoming []todo.Todo
	for _, t := range todos {
		if !listedUUIDs[t.UUID] {
			incoming = append(incoming, t)
		}
	}
	return incoming, nil
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

			activeName, _, _ := blisscontext.FindContext(cwd)
			personalMode := activeName == ""

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			// ── header ───────────────────────────────────────────────────
			date := stMuted.Render(time.Now().Format("Mon Jan 02, 2006"))
			fmt.Println(stTitle.Render("∴ bliss status") + "  " + date)
			fmt.Println()

			// ── contexts ─────────────────────────────────────────────────
			contextNames, err := s.ListContextNames()
			if err != nil {
				return err
			}
			type ctxRow struct {
				name   string
				path   string
				active bool
			}
			var rows []ctxRow
			for _, name := range contextNames {
				path, _ := s.ReadContextMeta(name)
				rows = append(rows, ctxRow{name, path, name == activeName})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
			for _, r := range rows {
				counts := statusListCounts(s, r.name)
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

	incomingCount := statusIncomingCount(s, contextUUID)
	if incomingCount > 0 {
		counts = append(counts, listCount{"incoming", incomingCount})
	}

	return counts
}

func statusIncomingCount(s *store.Store, contextUUID string) int {
	todos, err := getIncomingTodos(s, contextUUID)
	if err != nil {
		return 0
	}
	return len(todos)
}

// isContextPathFresh checks whether a path still contains a .bliss-context pointing to contextName.
func isContextPathFresh(path, contextName string) bool {
	data, err := os.ReadFile(filepath.Join(path, ".bliss-context"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == contextName
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync the store with the remote (fetch, pull, push)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.Open()
			if err != nil {
				return err
			}

			pushed, pulled, err := s.Sync()
			if err != nil {
				return err
			}

			switch {
			case pushed > 0:
				word := "commits"
				if pushed == 1 {
					word = "commit"
				}
				fmt.Printf("%s %d %s.\n", stMuted.Render("Pushed"), pushed, word)
			case pulled > 0:
				word := "commits"
				if pulled == 1 {
					word = "commit"
				}
				fmt.Printf("%s %d %s.\n", stMuted.Render("Pulled"), pulled, word)
			default:
				fmt.Println(stMuted.Render("Already up to date."))
			}

			return nil
		},
	}
}

func historyCmd() *cobra.Command {
	var all bool
	var personal bool
	var contextFlag string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show history of changes",
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

			var contextName string
			if !personal && !all {
				contextName, err = resolveContextName(s, contextFlag, cwd)
				if err != nil {
					return err
				}
			}

			// ── header ────────────────────────────────────────────────────
			date := stMuted.Render(time.Now().Format("Mon Jan 02, 2006"))
			if all {
				fmt.Println(stTitle.Render("∴ bliss history --all") + "  " + date)
			} else if personal || contextName == "" {
				fmt.Println(stTitle.Render("∴ bliss history") + "  " + stMuted.Render("Personal") + "  " + date)
			} else {
				ctxPath, _ := s.ReadContextMeta(contextName)
				fmt.Println(stTitle.Render("∴ bliss history") + "  " +
					stMuted.Render("Context:") + " " + stBold.Render(contextName) +
					"  " + stMuted.Render("Path:") + " " + stPath.Render(shortenHomePath(ctxPath)) +
					"  " + date)
			}
			fmt.Println()

			// ── entries ───────────────────────────────────────────────────
			entries, err := s.ReadHistory()
			if err != nil {
				return err
			}

			// Filter entries to the relevant scope.
			var filtered []store.HistoryEntry
			for _, e := range entries {
				switch {
				case all:
					filtered = append(filtered, e)
				case personal || contextName == "":
					if e.Personal {
						filtered = append(filtered, e)
					}
				default:
					if e.ContextName == contextName {
						filtered = append(filtered, e)
					}
				}
			}

			if len(filtered) == 0 {
				fmt.Println(stMuted.Render("(no history)"))
				return nil
			}

			// For --all, compute label width from context names (slugs).
			labelWidth := 0
			if all {
				contextNames, _ := s.ListContextNames()
				for _, name := range contextNames {
					if len(name) > labelWidth {
						labelWidth = len(name)
					}
				}
				if labelWidth < len("personal") {
					labelWidth = len("personal")
				}
			}

			for _, e := range filtered {
				ts := stMuted.Render(e.Time.Format("2006-01-02 15:04"))
				msg := strings.TrimPrefix(strings.TrimSpace(e.Message), "bliss: ")
				if all {
					var label string
					if e.ContextName != "" {
						label = e.ContextName
					} else if e.Personal {
						label = "personal"
					}
					labelStyled := stMuted.Render(fmt.Sprintf("%-*s", labelWidth, label))
					fmt.Printf("%s  %s  %s\n", ts, labelStyled, msg)
				} else {
					fmt.Printf("%s  %s\n", ts, msg)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show history across all contexts and personal")
	cmd.Flags().BoolVarP(&personal, "personal", "p", false, "Show personal history")
	cmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Show history for a specific context by name")
	return cmd
}


// listLabelStyle returns a muted grey style for list name labels.
// List names are structural labels, not data — they should not draw attention.
func listLabelStyle(_ string) lipgloss.Style {
	return stMuted
}

// sortedCounts returns counts in semantic display order using listSortKey.
func sortedCounts(counts []listCount) []listCount {
	out := make([]listCount, len(counts))
	copy(out, counts)
	sort.Slice(out, func(i, j int) bool {
		pi, pj := listSortKey(out[i].name), listSortKey(out[j].name)
		if pi != pj {
			return pi < pj
		}
		return out[i].name < out[j].name
	})
	return out
}

// listSortKey returns a sort priority for semantic list ordering:
// today → this-week → next-week → later → custom → bugs → incoming.
func listSortKey(name string) int {
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
	case "incoming":
		return 9
	default:
		return 4 // custom lists after "later", before "bugs"
	}
}

// sortListNames sorts list names in semantic display order.
func sortListNames(names []string) []string {
	out := make([]string, len(names))
	copy(out, names)
	sort.Slice(out, func(i, j int) bool {
		ki, kj := listSortKey(out[i]), listSortKey(out[j])
		if ki != kj {
			return ki < kj
		}
		return out[i] < out[j]
	})
	return out
}

// listSectionDelim returns a section delimiter aligned with the number field.
// pos is the next item position: <10 → 1-digit zone (2 dashes), ≥10 → 2-digit (3 dashes).
// Pass pos=1 when there are no position numbers (--all mode).
func listSectionDelim(pos int, name string) string {
	var s string
	if pos >= 10 {
		s = " ───"
	} else {
		s = "  ──"
	}
	if name != "" {
		s += " " + name
	}
	return stMuted.Render(s)
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
