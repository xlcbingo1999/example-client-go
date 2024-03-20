helm install clunky-serval ./helm-configmap

# 不会安装应用(chart)到你的kubenetes集群中，只会渲染模板内容到控制台（用于测试）
helm install --debug --dry-run goodly-guppy ./helm-configmap

# 外部设置一个values.yaml中的内容
helm install --debug --dry-run solid-vulture ./helm-configmap --set favoriteDrink=slurm
