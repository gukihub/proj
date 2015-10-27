package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func update_proj(db *sql.DB, dir string) (int, string) {
	var err error
	var fi_buf os.FileInfo
	var fi_tab []os.FileInfo
	var f *os.File

	f, err = os.Open(dir)
	if err != nil {
		return 1, fmt.Sprintf("failed to open %s (%v)\n", dir, err)
	}

	fi_buf, err = f.Stat()
	if err != nil {
		return 1, fmt.Sprintf("failed to stat %s (%v)\n", dir, err)
	}

	if fi_buf.IsDir() == false {
		return 1, fmt.Sprintf("%s is not a directory\n", dir)
	}

	fi_tab, err = f.Readdir(0)
	if err != nil {
		return 1, fmt.Sprintf("failed to read %s (%v)\n", dir, err)
	}

	// database statements
	tx, err := db.Begin()
	if err != nil {
		return 1, fmt.Sprintf("%v\n", err)
	}

	req := `
	insert into projects(name, author, creation_date)
	values(?, ?, ?)
	`

	stmt, err := tx.Prepare(req)
	if err != nil {
		return 1, fmt.Sprintf("%v\n", err)
	}

	// loop on the folders
	for _, g := range fi_tab {
		if strings.HasPrefix(g.Name(), ".") {
			continue
		}
		if g.IsDir() == false {
			continue
		}
		// insert project in the db
		_, err = stmt.Exec(g.Name(), me.Uid,
			g.ModTime().Format("2006-01-02 15:04:05"))
		if err != nil {
			fmt.Printf("error creating project %s (%v)\n",
				g.Name(), err)
			continue
		}
	}

	stmt.Close()
	tx.Commit()

	return 0, ""
}
