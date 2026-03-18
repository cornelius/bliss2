# Bliss 2.0

Peace of mind with todos. Note them down when something occurs to you. Get them back to the top of your head when needed. With context. With the lowest effort possible.

## Concept

Maintain a database of todos as plain text files which can be version controlled and shared easily. Readable by humans and machines in the same way. Provide clients to work with these lists. Provide a protocol to store and sync it on a server and a distributed way. Store them with context and enough meta information to make sense of them forever. Have an intelligent way to resurface them when you are ready to work on them. Have a convenient way to manage them, prioritize.

Should serve as personal Kanban. As todo list for a project. As a reminder for chores. As brain dump for ideas.

Underlying principle: Minimize brain cycles spent on todos. Free your mind to think big. Have all the details ready when needed, only when needed.

## Prior art

* https://github.com/cornelius/bliss (focused on the UI side, but built with the same model in mind, more info at https://blog.cornelius-schumacher.de/2013/06/experimenting-with-user-interfaces-for.html)
* https://apps.kde.org/de/korganizer/ (traditional todo manager, tailored to my needs and my taste)
* https://trello.com (lists of list, excellent tool for personal Kanban)
* http://todotxt.org/ (similar philosophy)
* https://en.wikipedia.org/wiki/Getting_Things_Done (influential background)

## Design

### Format for storing todos

Use plain text to store todos. Must be easily version controllable. Changes should be tracable also by humans in a diff. When edited by multiple clients, merge conflicts should be minimized and easily be resolvable.

Needs to support context, i.e. a kind of tag to provide some meta information about where and when the todo is relevant.

Needs to support creation and updated information, so it's clear when things change.

Needs to support grouping, so I can make sub lists, or represent columns on a board.

Possible implementation: One line per todo. Research existing formats.

A hand-written TODO or TODO.md file with a list of bullet points should be a valid way to store todos. Maybe we need different format flavors.

### Storage backend

Version controlled syncable storage. Possible implementation git.

Should be possible to sync todos between different computers.

If doable, should support encryption to be able to share data without having to trust transport or servers used to store the data. Don't start with that, but don't take design decisions which prevent it for the future.

One other possible implementation could be my project happiness.

It should be supported to have multiple files with todos at different locations. It should be possible to aggregate them.

Idea to explore for the future: Use issue trackers as backend. Or make content of issue trackers available through Bliss 2.0.

### Capturing todos

Command line client to capture todos. Store context of working directory. So if a directory serves as container for a project, it should be able to capture todos for this project by just calling a simple command there.

There should be a central file in the home directory (following standard conventiosn such as .local), to register places where todos are captured. So it is possible to provide a global overview of todos.

There are cases, where a local TODO.md is a good way to capture todos, in other cases maybe a gloval file with a richer special format (see section about format) is better. Do we need to sync both in some way?

It also should be possible to have other input channels, e.g. a mobile app, so todos can be captured on the go. It should be possible to assign them to a project, so they end up in the right context.

A graphical app should of course also be possible and supported. Maybe a widget or helper which sits in the menu bar or something like that.

It could be nice to auto-tag todos based on their content, so that they are assigned to the right context.

### Retrieving todos

Todos should always be retrievable on request. By default in the context where it matters. Getting a list.

There also should be a global overview. A list of lists, maybe a board. A board could also be handy in a project context.

There also should be a way to let todos popup in the right context and right moment. Maybe as some kind of notification, but I don't want to see that as annoying thing that you have to silence, but as the magic help which gives you the support you just need. This would require some (artificial) intelligence. It also would need to keep environmental factors in mind, such as time or physical location, e.g. bubbling up the shopping list, when you are in the shop. This could be modeled as some sort of triggers. It could also be self-learning, so that you could give feedback, if a todo was surfaced at the right time and then the system would take this to improve for the next time. This might be much more than retrieving todos and warrant an own section.

Basic principle: Clients, format, and storage should be flexible and workable. Robust for manual and automatic handling. They should serve as building blocks and basic elements which can be orchestrated into bigger systems and extended by features in a very modular way.

### Prioritizing todos

It needs to be possible to easily order todos to reflect priorities.

It needs to be possible to groom todo lists, i.e. go through a list of todos and sort them in different other lists, e.g. to plan when to do them (example categories could be today, this week, next week, later).

List should have optional separators for micro grouping todos.

### Completing todos

There needs to be a way to check off todos which have been done.

Done todos should vanish from the view visible all the time to remove mental load associated with them. They should be still available in the history on request, so they can be tracked, processed, and checked, when necessary.

Completing a todo should be fund and satisfying experience. Like taking a piece of paper, crumbling it to a ball and throwing it into the bin. Or like popping a balloon. Or like exploding something. It should feel great to complete a todo.

It also should be possible to remove todos. Not sure if it matters to make a difference between completing and removing a todo. Maybe not. And removing a todo can also be satisfying.

### Sketch for command line client

Initialize a directory to store todos: `bliss init`. This sets up this directory and all subdirectories as context for todos.

If there are multiple contexts, i.e. a directory hierarchy, where contexts are set up on different levels, the most specific context should be used.

Store a todo: `bliss add Feed the penguins` (implicitly stores creation time, and context)

List all todos of a context: `bliss list`.

Not sure how to mark a todo as done. Possible options:

* Let `bliss list` show ids for each todo. Close them with `bliss done <id>`. Could be a hassle to type. Also what should be the id? If it's long such as an uuid, it's hard to type. If it's short such as a running number, are ids changing then when others are done or removed?
* Have an interactive mode, where todos can be completed by choosing them from the list and check them off, e.g. with cursor keys and enter or space.
* Have a graphical applications for that. Could mean context switching when adding and completing todos.
* Manually edit the file where todos are stored (this should always be possible as an additional option).

Grooming todos could be done as interactive mode on `bliss groom`. It would present all todos (maybe one after each other), and give an option to move it to one of multiple buckets. These buckets could be templates or user-defined or some default categories.

### Technical decisions

All command line clients should be implemented in Go, so that executable can easily be installed without dependencies. It also gives an efficient language to write performant clients.

There should be an Android app to capture todos.

### Technical considerations

Need to evaluate if these could be the right decisions.

* Git as storage backend
* Event sourcing to gather and aggregate todos
* A REST API to store todos on the server (an existing protocol such as WebDAV (could give compatibility with Nextcloud etc.))

### Extensions

Let's call them blisslets.
