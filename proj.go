package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"time"
)

var prog_name string = os.Args[0]
var version string = "1.0"

var projroot string = os.Getenv("HOME") + "/projects"
var dbfile string = projroot + "/.proj.db"

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
			fmt.Printf("%s: failed to stat %s (%v)\n", prog_name, dir, err)
		}
		return false
	}

	if fi_buf.IsDir() == false {
		if verbose == true {
			fmt.Printf("%s: %s is not a directory\n", prog_name, dir)
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

/* create the projects table in the db */
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

/* populate the projects table */
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
	if ! checkdir(projdir , false) {
		err := os.Mkdir(projdir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var v_flag *bool
	var db *sql.DB

	v_flag = flag.Bool("version", false, "display program version")

	flag.Parse()

	if *v_flag == true {
		fmt.Printf("%s version %s\n", prog_name, version)
		return
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	if ! checkdir(projroot, true) {
		os.Exit(1)
	}

	if os.Args[1] != "init" && ! FileExists(dbfile) {
		fmt.Printf("%s: %s not found, run %s init\n",
			prog_name, dbfile, prog_name)
		os.Exit(1)
	}

	db = opendb()

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
	case "init":
		initdb(db)
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
}
