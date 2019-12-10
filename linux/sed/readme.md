# Append Text

``` bash
# If you want to edit the file in-place
$ sed -i -e 's/^/prefix/' file

# If you want to create a new file
$ sed -e 's/^/prefix/' file > file.new
```

[reference](https://stackoverflow.com/questions/2099471/add-a-prefix-string-to-beginning-of-each-line)
