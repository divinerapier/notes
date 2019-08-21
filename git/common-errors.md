1. fatal: Unable to find remote helper for 'https'
```
原因：没有权限执行 git-remote-https
解决方案：找到这个文件所在目录 /usr/libexec/git-core，加入到PATH里头.
```
