import catalog.models.schema.v1 as ac
import catalog.models.schema.v1.trait as t
import catalog.models.schema.v1.workload as wl
import catalog.models.schema.v1.workload.container as c
import catalog.models.schema.v1.workload.container.probe as p
import catalog.models.schema.v1.workload.secret as sec
import catalog.models.schema.v1.workload.network as n
import catalog.models.schema.v1.monitoring as m

import test_update_1 as tu
import helloworld as oci_helloworld

local = tu.The_first_kcl_program
oci = oci_helloworld.The_first_kcl_program

# base.k declares reusable configurations for all stacks.
hello: ac.AppConfiguration {
    workload: wl.Service {
        containers: {
            "nginx": c.Container {
                image: "nginx:v1"
                # Run the following command as defined
                command: ["/bin/sh", "-c", "echo hi"]
                # Extra arguments append to command defined above
                args: ["/bin/sh", "-c", "echo hi"]
                env: {
                    # An environment variable of name "env1" and value "VALUE" will be set
                    "env1": "VALUE"
                    # An environment variable of name "env2" and value of the key "key" in the
                    # secret named "sec-name" will be set.
                    "env2": "secret://sec-name/key"
                }
                # Run the command "/bin/sh -c echo hi", as defined above, in the directory "/tmp"
                workingDir: "/tmp"
                resources: {
                    "cpu": "2"
                    "memory": "4Gi"
                }
                # Configure a HTTP readiness probe
                readinessProbe: p.Probe {
                    probeHandler: p.Http {
                        url: "http://localhost:80"
                    }
                    initialDelaySeconds: 10
                }
            }
        }
        secrets: {
            "basic-auth": sec.Secret {
                $type: "basic"
                data: {
                    "username": "admin"
                    "password": "******"
                }
            }
        }
        replicas: 2
        ports: [
            n.Port {
                port: 80
                targetPort: 8080
                public: True
            }
        ]
    }
}

hellocollaset: ac.AppConfiguration {
    workload: wl.Service {
        containers: {
            "nginx": c.Container {
                image: "nginx:v1"
                # Run the following command as defined
                command: ["/bin/sh", "-c", "echo hi"]
                # Extra arguments append to command defined above
                args: ["/bin/sh", "-c", "echo hi"]
                # Run the command "/bin/sh -c echo hi", as defined above, in the directory "/tmp"
                workingDir: "/tmp"
            }  
        }
        ports: [
            n.Port {
                port: 80
            }
        ]
    }
    opsRule: t.OpsRule {
        maxUnavailable: "30%"
    }
}

hellojob: ac.AppConfiguration {
    workload: wl.Job {
        containers: {
            "busybox": c.Container {
                image: "busybox:1.28"
                # Run the following command as defined
                command: ["/bin/sh", "-c", "echo hello"]
            }
        }
        # Run every hour.
        schedule: "0 * * * *"
    }
}
