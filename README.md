<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# 介绍

该项目为 `cert-manager` 的 webhook 插件, 用于对接华为云DNS服务, 实现自动化证书签发和续期.

# 使用说明

1. 修改 deploy/cert-manager-webhook-huawei 目录的的values.yaml, 安装 cert-manager-webhook-huawei
    ```bash
    $ helm upgrade -i cert-manager-webhook-huawei deploy/cert-manager-webhook-huawei -n cert-manager
    ```

2. 安装[reflector](https://github.com/EmberStack/kubernetes-reflector), 用于自动同步证书
    ```bash
    $ helm repo add emberstack https://emberstack.github.io/helm-charts
    $ helm repo update
    $ helm upgrade -i reflector emberstack/reflector -n kube-system
    ```

3. 修改 deploy/cert 目录的的values.yaml 安装证书
   ```bash
    $ helm upgrade -i example-cn deploy/cert -n cert-manager
   ```

   上述配置会尝试申请`*.example.cn`泛域名证书, 并且把证书名命名为`example-tls`并放置到`default`命名空间中, 然后`reflector`会自动把证书同步到其它命名空间中.
