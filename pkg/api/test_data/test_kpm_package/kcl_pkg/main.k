import sub
import k8s.api.core.v1 as k8core

schema SchemaInMainK:
    msg: str

schema_in_main_k = SchemaInMainK {
    msg='I am the instance of SchemaInMainK'
}

schema_in_sub_k = sub.SchemaInSubK {
    msg='I am the instance of SchemaInSubK'
}

schema_in_k8s = k8core.Pod {
    metadata.name = "web-app"
    spec.containers = [{
        name = "main-container"
        image = "nginx"
        ports = [{containerPort: 80}]
    }]
}