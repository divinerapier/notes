# MySQL Command Line Tool

## Install

``` sh
# get pip
$ curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py

# install to the use site
$ python get-pip.py --user

# install to all uses
$ python get-pip.py
```

## Usage

### Login

```
$ tldr mycli
Local data is older than two weeks, use --update to update it.


mycli

A command line client for MySQL that can do auto-completion and syntax highlighting.
More information: <https://mycli.net>.

- Connect to a local database on port 3306, using the current user's username:
    mycli database_name

- Connect to a database (user will be prompted for a password):
    mycli -u username database_name

- Connect to a database on another host:
    mycli -h database_host -P port -u username database_name
```

### Command

#### Help

``` mycli
> help
```
or
``` mycli
> ?
```

``` 
mysql use@host:(none)> ?
+-------------+----------------------------+------------------------------------------------------------+
| Command     | Shortcut                   | Description                                                |
+-------------+----------------------------+------------------------------------------------------------+
| \G          | \G                         | Display current query results vertically.                  |
| \dt         | \dt[+] [table]             | List or describe tables.                                   |
| \e          | \e                         | Edit command with editor (uses $EDITOR).                   |
| \f          | \f [name [args..]]         | List or execute favorite queries.                          |
| \fd         | \fd [name]                 | Delete a favorite query.                                   |
| \fs         | \fs name query             | Save a favorite query.                                     |
| \l          | \l                         | List databases.                                            |
| \once       | \o [-o] filename           | Append next result to an output file (overwrite using -o). |
| \timing     | \t                         | Toggle timing of commands.                                 |
| connect     | \r                         | Reconnect to the database. Optional database argument.     |
| exit        | \q                         | Exit.                                                      |
| help        | \?                         | Show this help.                                            |
| nopager     | \n                         | Disable pager, print to stdout.                            |
| notee       | notee                      | Stop writing results to an output file.                    |
| pager       | \P [command]               | Set PAGER. Print the query results via PAGER.              |
| prompt      | \R                         | Change prompt format.                                      |
| quit        | \q                         | Quit.                                                      |
| rehash      | \#                         | Refresh auto-completions.                                  |
| source      | \. filename                | Execute commands from file.                                |
| status      | \s                         | Get status information from the server.                    |
| system      | system [command]           | Execute a system shell commmand.                           |
| tableformat | \T                         | Change the table format used to output results.            |
| tee         | tee [-o] filename          | Append all results to an output file (overwrite using -o). |
| use         | \u                         | Change to a new database.                                  |
| watch       | watch [seconds] [-c] query | Executes the query every [seconds] seconds (by default 5). |
+-------------+----------------------------+------------------------------------------------------------+
```

#### Set Pager

``` mycli
-- Set PAGER. Print the query results via PAGER.
> pager less -S
```
