resource "helm_release" "mysql" {
  namespace = kubernetes_namespace.fortune.id

  name  = "mysql"
  chart = "./charts/mysql"

  set {
    name  = "mysql.rootPassword"
    value = "example"
  }

  set {
    name  = "mysql.database"
    value = "fortune_db"
  }

  set {
    name  = "mysql.user"
    value = "kinduser"
  }

  set {
    name  = "mysql.password"
    value = "kindpassword"
  }
}
