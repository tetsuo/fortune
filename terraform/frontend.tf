resource "helm_release" "frontend" {
  name      = "frontend"
  namespace = kubernetes_namespace.fortune.id
  chart     = "./charts/frontend"

  values = [
    <<EOF
replicaCount: 1

frontend:
  port: 8080
  debugPort: 8081
  image:
    name: "${var.frontend_image_name}"
    tag: "${var.frontend_image_version}"
  containerName: "${var.frontend_container_name}"
  fqdn: "${var.frontend_fqdn}"
  logLevel: "debug"
  clusterEnv: "kind"

database:
  host: "${helm_release.mysql.metadata[0].name}.${helm_release.mysql.metadata[0].namespace}.svc.cluster.local"
  port: "3306"
  user: "kinduser"
  password: "kindpassword" # This is not safe; do not hardcode passwords.
  name: "fortune_db"

ingress:
  enabled: true
  class: "haproxy"
EOF
  ]
}
