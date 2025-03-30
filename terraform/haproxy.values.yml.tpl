controller:
  ingressClass: haproxy
  service:
    type: NodePort
    externalTrafficPolicy: Cluster
  replicaCount: 1
  autoscaling:
    minReplicas: 1
    maxReplicas: 1
  resources:
    requests:
      cpu: 40m
      memory: 64Mi
defaultBackend:
  resources:
    requests:
      cpu: 4m
  replicaCount: 1
  autoscaling:
    minReplicas: 1
    maxReplicas: 1
