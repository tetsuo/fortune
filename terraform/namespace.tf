resource "kubernetes_namespace" "fortune" {
  metadata {
    name = "fortune"
  }
}
