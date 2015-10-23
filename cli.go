package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shavac/readline"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

func banner() {
	fmt.Println(`                                    _/
     _/_/_/    _/  _/_/    _/_/
    _/    _/  _/_/      _/    _/  _/
   _/    _/  _/        _/    _/  _/
  _/_/_/    _/          _/_/    _/
 _/                            _/
_/                          _/           `)
	fmt.Println("       " + prog_name + " version " + version)
	fmt.Println("")
	fmt.Println("Welcome " + me.Name + " !")
}

func projcli(db *sql.DB) {
	var arg1 string
	var prompt string = "proj> "

	signal.Ignore(os.Interrupt, os.Kill)

	// loop until ReadLine returns nil (signalling EOF)
L:
	for {
		result := readline.ReadLine(&prompt)

		// exit loop with EOF(^D)
		if result == nil {
			println()
			//break L
			continue
		}

		r := strings.Split(*result, " ")

		if len(r) > 1 {
			arg1 = r[1]
		} else {
			arg1 = ""
		}

		switch r[0] {
		// ignore blank lines
		case "":
			continue L
		case "ls":
			listdb(db, arg1)
		case "ll":
			out, err := exec.Command("ls", "-ltr").Output()
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("%s", out)
			//ls(".")
		case "new":
			newproj(db, arg1)
		case "cd":
			// trim the project name in case of completion
			enterproj(strings.Trim(arg1, "/"))
		case "rm":
			rmproj(db, arg1)
		case "help":
			helpstr := prog_name + " help:" + `

  Command              Description
  ===================  ============================================
  ls [pattern]         list projects
  ll                   list folders in the projects directory
  new <project name>   create a new project
  cd <project name>    enter a project
  rm <project name>    delete a project from the database
  help                 print this help
  quit                 exit` + " " + prog_name + `
                        `
			fmt.Println(helpstr)
		case "quit":
			break L
		default:
			println("not found: " + *result)
		}

		readline.AddHistory(*result)
	}
}

func listdb(db *sql.DB, pattern string) {
	var req string
	datelayout := "2006-01-02 15:04:05"

	if pattern != "" {
		req = `
                select
                        projects.id, projects.name,
                        projects.creation_date, users.username
                from
                        projects, users
                where
                        projects.name like ` + "'%" + pattern + "%'" + `
                        and projects.author = users.id
                `
	} else {
		req = `
                select
                        projects.id, projects.name,
                        projects.creation_date, users.username
                from
                        projects, users
                where
                        projects.author = users.id
                `

	}
	rows, err := db.Query(req)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id int
		var name string
		var date string
		var username string
		rows.Scan(&id, &name, &date, &username)
		creation_date, _ := time.Parse(datelayout, date)

		/* list with user id
		   fmt.Printf("%s%03d%s %s %s[%s]%s %s%s%s\n",
		           "\033[31;1m",
		           id,
		           "\033[0m",
		           username,
		           "\033[43;31m",
		           creation_date.Format("2006-01-02"),
		           "\033[0m",
		           "\033[32;1m",
		           name,
		           "\033[0m")
		*/

		fmt.Printf("%s%03d%s %s[%s]%s %s%s%s\n",
			"\033[31;1m",
			id,
			"\033[0m",
			"\033[43;31m",
			creation_date.Format("2006-01-02"),
			"\033[0m",
			"\033[32;1m",
			name,
			"\033[0m")
	}
	rows.Close()
}

func newproj(db *sql.DB, name string) int {
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return 1
	}

	req := `
        insert into projects(name, author, creation_date)
        values(?, ?, datetime('now'))
        `

	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Println(err)
		return 1
	}

	_, err = stmt.Exec(name, me.Uid)
	if err != nil {
		log.Println(err)
		stmt.Close()
		tx.Commit()
		return 1
	}

	stmt.Close()
	tx.Commit()

	print("created project " + name + " in database")

	projdir := projroot + "/" + name

	if checkdir(projdir, false) {
		print("not creating directory " + projdir +
			" (already exist)")
		return 1
	}

	err = os.Mkdir(projdir, 0700)
	if err != nil {
		log.Println(err)
		return 1
	}

	print("created directory " + projdir)

	return 0
}

func rmproj(db *sql.DB, name string) int {
	projname := strings.Trim(name, "/")

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return 1
	}

	req := `
        select author from projects
        where name = '` + projname + "'"
	rows, err := db.Query(req)
	if err != nil {
		log.Println(err)
		return 1
	}
	var uid string
	for rows.Next() {
		rows.Scan(&uid)
	}
	rows.Close()
	if uid != me.Uid {
		fmt.Println("error: " + projname + " is not your project")
		return 1
	}

	req = `
        delete from projects
        where name = ?
        `

	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Println(err)
		return 1
	}

	_, err = stmt.Exec(projname)

	stmt.Close()
	tx.Commit()

	// return if the exec failed
	if err != nil {
		log.Println(err)
		return 1
	}

	print("deleted project " + projname + " from database")

	return 0
}

func enterproj(name string) {
	var err error
	var workdir string

	workdir, err = os.Getwd()
	if err != nil {
		log.Println(err)
	}

	print("Entering " + name + " project.")

	err = os.Chdir(projroot + "/" + name)
	if err != nil {
		log.Println(err)
		return
	}

	os.Setenv("PS1", "proj/"+name+"> ")
	cmd := exec.Command(projshell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}

	err = os.Chdir(workdir)
	if err != nil {
		log.Println(err)
		return
	}
	print("Leaving " + name + " project.")
}
