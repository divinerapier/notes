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

## Install Manjaro

### Download And Install Arch

从 [ArchWSL](https://github.com/yuk7/ArchWSL/releases) 下载 `Arch.zip`，解压到虚拟机期望存在的位置。

双击 `Arch.exe` 进行安装。

### Configure Arch WSL2

#### Create User

在 `Arch` 中运行

``` bash
sudo useradd -r -m -s /bin/bash your_name
sudo chmod +w /etc/sudoers
sudo vim /etc/sudoers
# 添加 your_name ALL=(ALL) ALL
sudo chmod -w /etc/sudoers

# 设置密码
passwd your_name
```

#### Set Default  WSL

``` powershell
# 设置运行时默认的用户
.\Arch.exe config --default-user your_name
# 设置默认运行的 WSL
#    --set-default、-s <发行版本>
#        将分发设置为默认值。
wsl -s Arch
```

### Mirrors

手动更改源排名

``` bash
sudo pacman-mirrors -i -c China -m rank
```

#### archlinux

编辑 `/etc/pacman.d/mirrorlist`， 在文件的最顶端添加：

``` conf
Server = https://mirrors.tuna.tsinghua.edu.cn/archlinux/$repo/os/$arch
```
更新软件包缓存:
``` bash
sudo pacman -Syy
```

[archlinux](https://mirrors.tuna.tsinghua.edu.cn/help/archlinux/)

[wiki](https://wiki.archlinux.org/index.php/Arch_Linux_Archive)

#### archlinuxcn

在 `/etc/pacman.conf` 文件末尾添加以下两行：

``` conf
[archlinuxcn]
Server = https://mirrors.tuna.tsinghua.edu.cn/archlinuxcn/$arch
```

之后安装 `archlinuxcn-keyring` 包导入 GPG key。

[archlinuxcn](https://mirrors.tuna.tsinghua.edu.cn/help/archlinuxcn/)

### Set Pacman Key

``` bash
pacman-key --init
pacman-key -S manjaro-keyring
```

### Install yaourt

``` bash
sudo pacman -Syy yaourt
```

[Yaourt 已死！在 Arch 上使用这些替代品](https://zhuanlan.zhihu.com/p/42287487)

### Install yay

``` bash
git clone https://aur.archlinux.org/yay.git
cd yay
makepkg -si
```

#### Configure yay

执行以下命令修改 aururl :

``` bash
yay --aururl “https://aur.tuna.tsinghua.edu.cn” --save
```

修改的配置文件

``` bash
vim ~/.config/yay/config.json
```

查看配置

``` bash
yay -P -g
```

### Install Specific Version of Package Via Pacman

从 [archlinux packages](https://archive.archlinux.org/packages) 找到目标 package。

比如，`clion` 要求 `cmake` 版本在 `2.8.11` 到 `3.16.x` 之间，而 `pacman` 默认版本为 `3.17`。 从上面的链接找到一个合适的版本，执行

``` bash
sudo pacman -U http://archive.archlinux.org/packages/c/cmake/cmake-3.16.2-1-x86_64.pkg.tar.xz
```

https://forum.manjaro.org/t/how-to-install-specific-version-of-package-via-pacman/50668

## Install GUI

### X410

一款 `x-server` 软件，在 `Microsoft Store` 下载。

### Linux ENV

``` bash
# for x11
export DISPLAY=$(awk '/nameserver / {print $2; exit}' /etc/resolv.conf 2>/dev/null):0
export XDG_RUNTIME_DIR=/home/divinerapier
export RUNLEVEL=3
export $(dbus-launch)
export LIBGL_ALWAYS_INDIRECT=1

# for locale
export LC_CTYPE=zh_CN.UTF-8
export LANG="zh_CN.UTF-8"
export LC_ALL="zh_CN.UTF-8"
```

[how-to-export-dbus-session-bus-address](https://stackoverflow.com/questions/41242460/how-to-export-dbus-session-bus-address)

[how-to-set-up-working-x11-forwarding-on-wsl2](https://stackoverflow.com/questions/61110603/how-to-set-up-working-x11-forwarding-on-wsl2)

### Chinese Font

``` bash
sudo pacman -S wqy-microhei
```

### Locale

执行

``` bash
sudo vim /etc/locale.gen
```

取消下面两行的注释

``` conf
en_US.UTF-8 UTF-8
zh_CN.UTF-8 UTF-8
```

初始化语言环境

``` bash
sudo locale-gen
```

https://my.oschina.net/u/4362704/blog/3308054

### XFCE4

#### Install XFCE4

``` bash
sudo pacman -S xfce4 xfce4-terminal
```

#### Start XFCE4

``` bash
startxfce4
```

### KDE

#### Install KDE

``` bash
sudo pacman -S plasma-meta
sudo pacman -S xorg
sudo pacman -S plasma-wayland-session
sudo pacman -S plasma kio-extras
sudo pacman -S kde-applications
```

#### Start KDE

``` bash
startplasma-x11 --all
```

## WSL2 Settings

配置文件位于 `%UserProfile%/.wslconfig`

``` ini
[wsl2]
kernel=<path>              # An absolute Windows path to a custom Linux kernel.
memory=<size>              # How much memory to assign to the WSL2 VM.
processors=<number>        # How many processors to assign to the WSL2 VM.
swap=<size>                # How much swap space to add to the WSL2 VM. 0 for no swap file.
swapFile=<path>            # An absolute Windows path to the swap vhd.
localhostForwarding=<bool> # Boolean specifying if ports bound to wildcard or localhost in the WSL2 VM should be connectable from the host via localhost:port (default true).

# <path> entries must be absolute Windows paths with escaped backslashes, for example C:\\Users\\Ben\\kernel
# <size> entries must be size followed by unit, for example 8GB or 512MB
```

[Introduce %UserProfile%/.wslconfig file for tweaking WSL2 settings](https://docs.microsoft.com/en-us/windows/wsl/release-notes#build-18945)

### Use Windows Chrome

``` bash
sudo ln -s /mnt/c/Program\ Files\ \(x86\)/Google/Chrome/Application/chrome.exe /usr/bin/chrome
```

[wsl-webbrowser](https://www.smslit.top/2017/09/02/wsl-webbrowser/)

## Clion

### Install In WSL2

``` bash
yay -S clion
yay -S archlinuxcn/clion-cmake
```

### Connect To WSL From Windows

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
