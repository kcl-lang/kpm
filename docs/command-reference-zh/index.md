# kpm

kpm 命令。

## 使用

```shell
kpm  <command> [arguments]...
```

## 介绍

`kpm` 是 kcl 包管理工具。它用于分发和管理 kcl 包。

## 选项

### --help, -h

展示 `kpm` 命令的帮助信息。

### --version, -v

展示 `kpm` 命令的版本信息。

## 子命令

- [kpm init](./1.init.md) - 初始化一个 kcl 包
- [kpm add](./2.add.md) - 添加一个依赖到 kcl 包
- [kpm pkg](./3.pkg.md) - 打包一个 kcl 包为 `*.tar`
- [kpm metadata](./4.metadata.md) - 打印一个 kcl 包的元数据
- [kpm run](./5.run.md) - 编译一个 kcl 包为 yaml 并运行
- [kpm login](./6.login.md) - 登录到一个 kcl registry
- [kpm logout](./7.logout.md) - 登出一个 kcl registry
- [kpm push](./8.push.md) - 上传一个 kcl 包到一个 registry
- [kpm pull](./9.pull.md) - 下载一个 kcl 包从一个 registry
- [kpm help](./10.help.md) - 打印 kpm 命令的帮助信息
