/*
 *  proj - manage your projects
 *
 *  Guillaume Kielwasser
 *
 *  2015/10/21
 *
 */

package main

import (
	"database/sql"
	"flag"
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

const version string = "1.0"

var (
	prog_name string = os.Args[0]

	projroot string = os.Getenv("HOME") + "/projects"
	dbfile   string = projroot + "/.proj.db"

	projshell string = "/usr/bin/ksh"
)

type Person struct {
	key       sql.NullInt64
	firstName sql.NullString
	lastName  sql.NullString
	address   sql.NullString
	city      sql.NullString
}

type employee struct {
	empID       sql.NullInt64
	empName     sql.NullString
	empAge      sql.NullInt64
	empPersonId sql.NullInt64
}

func checkdir(dir string, verbose bool) bool {
	f, err := os.Open(dir)

	if err != nil {
		if verbose == true {
			fmt.Printf("%s: %v\n", prog_name, err)
		}
		return false
	}

	fi_buf, err := f.Stat()

	if err != nil {
		if verbose == true {
			fmt.Printf("%s: failed to stat %s (%v)\n",
				prog_name, dir, err)
		}
		return false
	}

	if fi_buf.IsDir() == false {
		if verbose == true {
			fmt.Printf("%s: %s is not a directory\n",
				prog_name, dir)
		}
		return false
	}

	return true
}

func FileExists(fn string) bool {
	if _, err := os.Stat(fn); err == nil {
		return true
	} else {
		return false
	}
}

func opendb() *sql.DB {
	var db *sql.DB
	var err error

	db, err = sql.Open("sqlite3", dbfile)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func closedb(db *sql.DB) {
	db.Close()
}

// create the projects table in the db
func initdb(db *sql.DB) {
	sqlStmt := `
	create table projects (
		id integer not null primary key,
		name text,
		author integer,
		creation_date text
	);
	delete from projects;
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}

// populate the projects table
func popdb(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	req := `
	insert into projects(name, author, creation_date)
	values(?, ?, datetime('now'))
	`
	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(fmt.Sprintf("project%03d", i), 1)
		if err != nil {
			log.Fatal(err)
		}
	}
	stmt.Close()
	tx.Commit()
}

func listdb(db *sql.DB, pattern string) {
	var req string
	datelayout := "2006-01-02 15:04:05"

	if pattern != "" {
		req = `
		select
			id, name, creation_date
		from
			projects
		where
			name like ` + "'%" + pattern + "%'"
	} else {
		req = `
		select
			id, name, creation_date
		from
			projects
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
		rows.Scan(&id, &name, &date)
		creation_date, _ := time.Parse(datelayout, date)
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

func usage() {
	fmt.Printf("usage: %s ls [pattern]|new <project name>|init\n",
		prog_name)
}

func newproj(db *sql.DB, name string) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	req := `
	insert into projects(name, author, creation_date)
	values(?, ?, datetime('now'))
	`
	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(name, 1)
	if err != nil {
		log.Fatal(err)
	}
	stmt.Close()
	tx.Commit()

	projdir := projroot + "/" + name
	if !checkdir(projdir, false) {
		err := os.Mkdir(projdir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func enterproj(name string) {
	var err error
	var workdir string

	workdir, err = os.Getwd()
	if err != nil {
		log.Println(err)
	}

	fmt.Println("Entering " + name + " project.")

	err = os.Chdir(projroot + "/" + name)
	if err != nil {
		log.Println(err)
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
	}
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
			ls(".")
		case "new":
			newproj(db, arg1)
		case "cd":
			// trim the project name in case of completion
			enterproj(strings.Trim(arg1, "/"))
		case "help":
			helpstr := prog_name + " help:" + `

  Command              Description
  ===================  ============================================
  ls [pattern]         list projects
  ll                   list folders in the projects directory
  new <project name>   create a new project
  cd <project name>    enter a project
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

func main() {
	var v_flag *bool
	var h_flag *bool
	var db *sql.DB
	var err error

	v_flag = flag.Bool("version", false, "display program version")
	h_flag = flag.Bool("help", false, "display "+prog_name+" usage")

	flag.Parse()

	if *v_flag == true {
		fmt.Printf("%s version %s\n", prog_name, version)
		return
	}
	if *h_flag == true {
		usage()
		return
	}

	// check the projects root folder exists
	if !checkdir(projroot, true) {
		os.Exit(1)
	}

	// enter the projects root folder
	err = os.Chdir(projroot)
	if err != nil {
		log.Fatal(err)
	}

	// case we initialize the database
	if len(os.Args) == 2 && os.Args[1] == "init" {
		if !FileExists(dbfile) {
			db = opendb()
			initdb(db)
			closedb(db)
			return
		} else {
			fmt.Printf("%s: %s exists!\n", prog_name, dbfile)
			os.Exit(1)
		}
	}

	// check the db exists
	if !FileExists(dbfile) {
		fmt.Printf("%s: %s not found, run %s init\n",
			prog_name, dbfile, prog_name)
		os.Exit(1)
	}

	db = opendb()

	// with no args, run projcli
	if len(os.Args) == 1 {
		projcli(db)
		return
	}

	// parse arguments
	switch os.Args[1] {
	case "ls":
		var pattern string
		if len(os.Args) == 3 {
			pattern = os.Args[2]
		} else {
			pattern = ""
		}
		listdb(db, pattern)
	case "pop":
		popdb(db)
	case "new":
		if len(os.Args) != 3 {
			usage()
			os.Exit(1)
		}
		newproj(db, os.Args[2])
	default:
		usage()
	}

	closedb(db)

	return
}
