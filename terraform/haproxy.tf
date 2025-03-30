# The file contains the HAproxy ingress resource based on the official Helm chart (v1.22).
# Note that, a null resource is accompanied for patching the config map for some additional changes.
#
# - https://github.com/haproxytech/helm-charts/tree/main/kubernetes-ingress
# - https://www.haproxy.com/documentation/hapee/latest/onepage/

locals {
  haproxy_config = templatefile("${path.module}/haproxy.values.yml.tpl", {
  })
}

resource "helm_release" "haproxy" {
  name       = "haproxy"
  version    = var.haproxy_version
  chart      = "kubernetes-ingress"
  repository = "https://haproxytech.github.io/helm-charts"

  namespace = "ingress-controller"

  create_namespace = true

  values = [
    local.haproxy_config
  ]
}
