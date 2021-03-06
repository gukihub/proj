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
	//fmt.Println("Welcome " + me.Name + " !")
}

func clihelp() {
	helpstr := `
  Command              Description
  ===================  ===============================================
  ls [regexp]          list projects, optionally filter with a regexp
  ll                   list folders in the projects directory
  new <project name>   create a new project
  cd <project name>    enter a project
  rm <project name>    delete a project from the database
  du                   display projects disk usage
  find [pattern]       find files for pattern
  help                 print this help
  quit                 exit` + " " + prog_name + `
                        `
	fmt.Println(helpstr)
}

func runcmd(cmd string) {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("%s", out)
}

func projcli(db *sql.DB) {
	var arg1 string

	signal.Ignore(os.Interrupt, os.Kill)

	// loop until ReadLine returns nil (signalling EOF)
L:
	for {
		result := readline.ReadLine(&cliprompt)

		// exit loop with EOF(^D)
		if result == nil {
			println()
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
			//ls(".")
			runcmd("ls -ltr")
		case "new":
			newproj(db, arg1)
		case "cd":
			// trim the project name in case of completion
			enterproj(strings.Trim(arg1, "/"))
		case "rm":
			rmproj(db, arg1)
		case "du":
			runcmd("du -sk * | sort -n")
		case "find":
			format := "find * -type f -exec grep -l \"%s\" {} \\;"
			cmd := fmt.Sprintf(format, arg1)
			runcmd(cmd)
		case "help":
			clihelp()
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
			projects.name regexp ` + "'" + pattern + "'" + `
                        and projects.author = users.id
		order by
			projects.creation_date
                `
		//projects.name glob ` + "'*" + pattern + "*'" + `
	} else {
		req = `
                select
                        projects.id, projects.name,
                        projects.creation_date, users.username
                from
                        projects, users
                where
                        projects.author = users.id
		order by
			projects.creation_date
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

func filedate(file string) string {
	var fi_buf os.FileInfo
	var f *os.File
	var epoch int64
	var err error

	f, err = os.Open(file)
	if err != nil {
		return ""
	}
	fi_buf, err = f.Stat()
	if err != nil {
		return ""
	}
	epoch = fi_buf.ModTime().Unix()
	return time.Unix(epoch, 0).Format("2006-01-02 15:04:05")
}

func newproj(db *sql.DB, name string) int {
	var creation_date string

	projdir := projroot + "/" + name

	if checkdir(projdir, false) {
		creation_date = filedate(projdir)
	} else {
		creation_date = time.Now().Format("2006-01-02 15:04:05")
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return 1
	}

	req := `
        insert into projects(name, author, creation_date)
        values(?, ?, ?)
        `

	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Println(err)
		return 1
	}

	_, err = stmt.Exec(name, me.Uid, creation_date)
	if err != nil {
		log.Println(err)
		stmt.Close()
		tx.Commit()
		return 1
	}

	stmt.Close()
	tx.Commit()

	print("created project " + name + " in database")

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
