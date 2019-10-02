``` bash
$ git filter-branch -f --commit-filter 'git commit-tree -S "$@"' HEAD
```
