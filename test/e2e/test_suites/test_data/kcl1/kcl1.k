import konfig.base.pkg.kusion_kubernetes.api.apps.v1 as apps

demo = apps.Deployment {
    metadata.name = "nginx-deployment"
    spec = {
        replicas = 3
        selector.matchLabels = {
            app = "nginx"
        }
        template.metadata.labels = {
            app = "nginx"
        }
        template.spec.containers = [
            {
                name = "nginx"
                image = "nginx:1.14.2"
                ports = [
                    {containerPort = 80}
                ]
            }
        ]
    }
}