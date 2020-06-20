# WSL

## Install WSL

``` powershell
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
```

## Install WSL2

### Update Windows

若要更新到 WSL 2，必须满足以下条件：
* 运行 Windows 10（[已更新到版本 2004](https://docs.microsoft.com/zh-cn/windows/wsl/install-win10) 的内部版本 19041 或更高版本）。
* 通过按 Windows 徽标键 + R，检查你的 Windows 版本，然后键入 winver，选择“确定”。 （或者在 Windows 命令提示符下输入 ver 命令）。 如果内部版本低于 19041，请[更新到最新的 Windows 版本](https://docs.microsoft.com/zh-cn/windows/wsl/install-win10)。 [获取 Windows 更新助手](https://docs.microsoft.com/zh-cn/windows/wsl/install-win10)。

### Enable the 'Virtual Machine Platform' optional component

安装 WSL 2 之前，必须启用“虚拟机平台”可选功能。

以管理员身份打开 PowerShell 并运行：

``` powershell
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
```

**重新启动计算机**，以完成 WSL 安装并更新到 WSL 2。

### Set WSL 2 as your default version

安装新的 Linux 分发版时，请在 Powershell 中运行以下命令，以将 WSL 2 设置为默认版本：

``` powershell
wsl --set-default-version 2
```

## Install Linux From Microsoft Store

1. 打开 [Microsoft Store](https://aka.ms/wslstore)，并选择你偏好的 Linux 分发版。
2. 在分发版的页面中，选择“获取”。
3. 启动 `Linux` 发行版，等待提示，设置用户名和密码。

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
