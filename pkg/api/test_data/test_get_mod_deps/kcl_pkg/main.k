import sub
import sub.sub1 as s1
import k8s.api.core.v1 as k8core

schema SchemaInMainK:
    msg: str

schema SchemaWithSameName:
    msg: str

schema_in_main_k = SchemaInMainK {
    msg='I am the instance of SchemaInMainK'
}

schema_in_sub_k = sub.SchemaInSubK {
    msg='I am the instance of SchemaInSubK'
}

schema_with_same_name = SchemaWithSameName {
    msg='I am the instance of SchemaWithSameName in main.k'
}

schema_with_same_name_in_sub = s1.SchemaWithSameName {
    msg='I am the instance of SchemaWithSameName in sub.k'
}

schema_in_k8s = k8core.Pod {
    metadata.name = "web-app"
    spec.containers = [{
        name = "main-container"
        image = "nginx"
        ports = [{containerPort: 80}]
    }]
}