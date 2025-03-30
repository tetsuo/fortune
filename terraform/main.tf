provider "kubernetes" {
  config_path    = "${path.module}/kubeconfig.yaml" # kind get kubeconfig --name kind > kubeconfig.yaml
  config_context = "kind-kind"
}

provider "helm" {
  kubernetes {
    config_path    = "${path.module}/kubeconfig.yaml"
    config_context = "kind-kind"
  }
}
