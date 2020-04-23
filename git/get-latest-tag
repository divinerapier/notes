# Get Latest Tag

## Usage

``` bash
# The command finds the most recent tag that is reachable from a commit.
# If the tag points to the commit, then only the tag is shown.
# Otherwise, it suffixes the tag name with the number of additional commits on top of the tagged object 
# and the abbreviated object name of the most recent commit.
git describe
```

``` bash
# With --abbrev set to 0, the command can be used to find the closest tagname without any suffix:
git describe --abbrev=0
```

``` bash
# other examples
git describe --abbrev=0 --tags # gets tag from current branch
git describe --tags `git rev-list --tags --max-count=1` # gets tags across all branches, not just the current branch
```

## Reference

[get-latest-tag-on-git.sh](https://gist.github.com/rponte/fdc0724dd984088606b0)
