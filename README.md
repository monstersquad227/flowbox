# flowbox

> Tips
> 
> 1. 此项目基于Mac OS系统开发，M芯片，如果在Linux上是会报错的，需要更换项目 bin/ 文件夹中的内容
> 2. 此项目基于 Kubernetes v1.18.6 版本开发，如果 api 版本不一致则不会创建相应的资源

| 工具          | 版本     |
|-------------|--------|
| go          | 1.23最低 |
| kubebuilder | latest |
| docker      | latest |

```bash
# 初始化项目
mkdir flowbox && cd flowbox
go mod init flowbox
kubebuilder init --domain flowbox.io
# y 回车
kubebuilder create api --group devflow --version v1 --kind FlowBox
```

## 部署项目

将写好的 `internal/controller/flowbox_controller.go` 代码和 `api/v1/flowbox_types.go` 保存好

```bash
make manifests 
# 会将 controller 中的注解生成对应的 config/rbac/role.yaml
make docker-build docker-push IMG=harbor.xxx.com/base/flowbox:v1
# 会生成相应的镜像，并推送镜像仓库
make install 
# 会创建对应的资源
make deploy IMG=harbor.xxx.com/base/flowbox:v1
# 会部署该镜像
```
> Tips:
> 
> make 对应指令查看 Makefile

## 后续

添加 webhook，flowbox创建的资源不允许原生删除，只用通过flowbox删除和更新