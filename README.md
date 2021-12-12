# fix_log4j

用于修复Kubernetes集群环境中的Apache Log4j2高危漏洞的命令行工具

# 漏洞信息

[Apache Log4j2](https://github.com/apache/logging-log4j2) 是一款开源的 Java 日志记录工具，大量的业务框架都使用了该组件。此次漏洞是用于 Log4j2 提供的 lookup 功能造成的，该功能允许开发者通过一些协议去读取相应环境中的配置。但在实现的过程中，并未对输入进行严格的判断，从而造成漏洞的发生。

### 相关链接

* [腾讯容器安全服务首推Apache Log4j2漏洞线上修复方案](https://mp.weixin.qq.com/s/scvnfJl2hc0cUXnWygGO_w)


# 本工具使用

## 命令行方式

### 获取

* 下载源码到本地，需要golang(>=1.5)环境，进入目录执行`make build`，可执行程序会被编译到`./bundles/`目录下
* 从`COS`上下载已经编译好的可执行文件
  * [Linux amd64](https://tcss-compliance-1258344699.cos.ap-guangzhou.myqcloud.com/tools/fix_log4j2/v0.2.2/fix_log4j-linux-adm64.tar.gz)
  * [Mac OSX](https://tcss-compliance-1258344699.cos.ap-guangzhou.myqcloud.com/tools/fix_log4j2/v0.2.2/fix_log4j-darwin-adm64.tar.gz)

### 配置

配置示例

```yaml
main:
  # kubeConfig 可选，可使用环境变量 KUBECONFIG 来指定
  kubeConfig: /root/.kube/config

clue:
  # 需要处理的镜像列表
  images:
    - docker.io/vulfocus/log4j2-rce-2021-12-09:latest
    - docker.io/library/busybox@sha256:b5cfd4befc119a590ca1a81d6bb0fa1fb19f1fbebd0397f25fae164abe1e8a6a
    - ccr.ccs.tencentyun.com/yunding/tcss-agent:1.8.2109.20
```
现阶段，镜像名称需要填入完整的名称，比如使用了来自`Docker Hub`的`vulfocus/log4j2-rce-2021-12-09`镜像，需要完整的填入省略的`Host`和`Tag`部分，即`docker.io/vulfocus/log4j2-rce-2021-12-09:latest`

### 执行

`./bundles/fix_log4j -c ./config.yaml`

## Kubernetes Job 方式

执行`kubectl apply -f https://tcss-compliance-1258344699.cos.ap-guangzhou.myqcloud.com/tools/fix_log4j2/job.yaml`

# 工具执行逻辑

1. 对k8s集群中的所有Pod进行遍历，发现引用了包含风险镜像的容器，列出待处理的Pod列表
2. 根据待处理的Pod列表获取最上层调度资源，如`Deployment`,`DaemonSet`等
3. 对`Deployment`,`DaemonSet`等资源执行修复，即检查Pod.Spec 中的`Command`和`Args`，如果是`java`启动的，则加入`-Dlog4j2.formatMsgNoLookups=true`参数
4. 执行更新

# 风险和注意事项

1. 当前版本暂时只支持对`>=2.10+`的版本，更低版本暂不支持
2. 更新工作负载有失败的可能，需要关注工作负载的状态、注意对业务的影响

# TODO

* [ ] 支持较老版本的修复策略
* [ ] 支持增加环境变量 FORMAT_MESSAGES_PATTERN_DISABLE_LOOKUPS=true
* [ ] 支持更新后的检查，如果失败可自动回滚
* [ ] 支持镜像更新
* [ ] 支持自动检查，而不用指定具体镜像列表

# Q&A

Github 的 Issue
