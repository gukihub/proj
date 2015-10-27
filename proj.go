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
	"log"
	"os"
	"os/user"
)

const version string = "1.0"

var (
	prog_name string = os.Args[0]

	projroot   string = os.Getenv("HOME") + "/projects"
	dbfilename string = "/.proj.db"
	dbfile     string = projroot + dbfilename

	projshell string = "/usr/bin/ksh"

	verbose = 1

	me *user.User

	cliprompt string = "proj> "
)

func print(str string) {
	if verbose == 0 {
		return
	}

	fmt.Println(str)
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
	var sqlStmt string
	var err error

	sqlStmt = `
	create table projects (
		id integer not null primary key,
		name text unique not null,
		author integer not null,
		creation_date text not null
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	sqlStmt = `
	create table users (
		id integer not null primary key,
		username text unique not null,
		name text unique not null
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}

func usage() {
	fmt.Printf("usage: %s ls [pattern]|new <project name>|", prog_name)
	fmt.Printf("rm <project name>|init\n")
}

func reguser(db *sql.DB, me *user.User) int {
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return 1
	}
	req := `
        insert into users(id, username, name)
        values(?, ?, ?)
        `

	stmt, err := tx.Prepare(req)
	if err != nil {
		log.Println(err)
		return 1
	}

	_, err = stmt.Exec(me.Uid, me.Username, me.Name)

	stmt.Close()
	tx.Commit()

	// don't log anything here, it just means the user already exits
	if err != nil {
		return 1
	}
	return 0
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

	// who am i?
	me, err = user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// where is my projroot
	if os.Getenv("PROJ_HOME") != "" {
		projroot = os.Getenv("PROJ_HOME")
		dbfile = projroot + dbfilename
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
			// add existing projects to the db, if any
			rc, str := update_proj(db, ".")
			if rc != 0 {
				fmt.Println(str)
			}
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

	// register as a user in case of
	reguser(db, me)

	// with no args, run projcli
	if len(os.Args) == 1 {
		banner()
		projcli(db)
		return
	}

	verbose = 0

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
	case "new":
		if len(os.Args) != 3 {
			usage()
			os.Exit(1)
		}
		newproj(db, os.Args[2])
	case "rm":
		if len(os.Args) != 3 {
			usage()
			os.Exit(1)
		}
		rmproj(db, os.Args[2])
	default:
		usage()
	}

	closedb(db)

	return
}
