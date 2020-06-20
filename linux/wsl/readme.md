# WSL

## Install Ubuntu

``` powershell
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
```

## Clion

[how-to-use-wsl-development-environment-in-clion](https://www.jetbrains.com/help/clion/how-to-use-wsl-development-environment-in-clion.html)

[Using WSL toolchains in CLion on Windows](https://www.youtube.com/watch?v=xnwoCuHeHuY)

## Errors

### 0x800701bc

`WslRegisterDistribution failed with error: 0x800701bc`

``` powershell
PS C:\WINDOWS\system32> ubuntu2004
Installing, this may take a few minutes...
WslRegisterDistribution failed with error: 0x800701bc
Error: 0x800701bc WSL 2 ?????????????????? https://aka.ms/wsl2kernel

Press any key to continue...
```

https://github.com/microsoft/WSL/issues/5393

https://docs.microsoft.com/zh-cn/windows/wsl/wsl2-kernel
