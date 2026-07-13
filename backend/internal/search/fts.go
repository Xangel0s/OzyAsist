package search

import "github.com/ozyassist/backend/internal/db"

type Result = db.SearchResult

// GlobalSearch realiza búsqueda combinada en mensajes, chats, proyectos y memoria.
func GlobalSearch(query string, limit int) (messages, chats, projects, memory []Result, err error) {
	if limit <= 0 {
		limit = 10
	}

	messages, err = db.FTS5SearchMessages(query, limit)
	if err != nil {
		return
	}

	chats, err = db.SearchChats(query, limit)
	if err != nil {
		return
	}

	projects, err = db.SearchProjects(query, limit)
	if err != nil {
		return
	}

	memory, err = db.FTS5SearchMemory(query, limit)
	if err != nil {
		return
	}

	return
}
