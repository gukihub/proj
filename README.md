# proj - Manage your projects

`proj` helps you to keep track of your projects on Linux.

A project under `proj` is a folder and a creation date. What's inside the
folder is what makes your project.

`proj` keep the projects information in a SQLite database in the projects
root folder, so this folder can be shared.

You can list, search and enter projects. Under `proj`, entering a
project means running a shell in the project's folder.

## Getting started

`proj` is expecting the projects workspace to be set in the environment
variable `PROJ_HOME`. If not found it will use `$HOME/projects`.

### Create the projects workspace

If you already have a folder with your
projects just set `PROJ_HOME` to that folder.

```bash
$ mkdir $HOME/projects
```

### Initialize the database

If the projects folder is not empty, the `init`
process will import the projects in the database.

```bash
$ proj init
```

### Run `proj`

```bash
$ proj
                                    _/
     _/_/_/    _/  _/_/    _/_/
    _/    _/  _/_/      _/    _/  _/
   _/    _/  _/        _/    _/  _/
  _/_/_/    _/          _/_/    _/
 _/                            _/
_/                          _/           
       proj version 1.0

proj> 
```

### Create a project

Use the command `new <project name>` to create a new project. `proj` will
add the new project to its database and create the according folder under
the projects workspace.

```
proj> new projectx
created project projectx in database
created directory /home/guki/projects/projectx
```

### Enter a project

Use the command `cd <project name>` to enter the project. `proj` will open
a shell in the project's folder.

```
proj> cd projectx
Entering projectx project.
proj/projectx> 
```

To leave a project just exit the shell.

## Commands

To list the available commands, run `help` under `proj` prompt.

```
proj> help

  Command              Description
  ===================  ============================================
  ls [pattern]         list projects
  ll                   list folders in the projects directory
  new <project name>   create a new project
  cd <project name>    enter a project
  rm <project name>    delete a project from the database
  du                   display projects disk usage
  find [pattern]       find files for pattern
  help                 print this help
  quit                 exit /home/kielwgui/gowork/bin/proj

```

