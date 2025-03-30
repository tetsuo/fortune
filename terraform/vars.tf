variable "haproxy_version" {
  default = "1.44"
}

variable "frontend_port" {
  default = "8080"
}

variable "frontend_debug_port" {
  default = "8081"
}

variable "frontend_container_name" {
  default = "frontend-server"
}

variable "frontend_image_name" {
  default = "fortune-frontend"
}

variable "frontend_image_version" {
  default = "latest"
}

variable "frontend_fqdn" {
  default = "local.haproxy.kind"
}
