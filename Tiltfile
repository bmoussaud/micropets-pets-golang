LOCAL_PATH = os.getenv("LOCAL_PATH", default='.')
NAMESPACE = os.getenv("NAMESPACE", default='micropets-dev')

compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/pets -buildmode pie -trimpath ./cmd/pets/main.go'

k8s_yaml(["config/application-configuration.yaml"])

local_resource(
  'go-build',
  compile_cmd,
  deps=['./cmd', './service','./internal'],
  dir='.')

allow_k8s_contexts('aks-eu-tap-6')

#k8s_yaml(["config/serviceclaims-aria.yaml"])

k8s_custom_deploy(
    'pets',
    apply_cmd="tanzu apps workload apply -f config/workload.yaml --update-strategy replace --debug --live-update" +
              " --local-path " + LOCAL_PATH +
              " --namespace " + NAMESPACE +
              " --yes --output yaml",    
    delete_cmd="tanzu apps workload delete -f config/workload.yaml --namespace " + NAMESPACE + " --yes",
    deps=['./build'],
    container_selector='workload',
    live_update=[      
      sync('./build', '/tmp/tilt')  ,      
      run('cp -rf /tmp/tilt/* /layers/tanzu-buildpacks_go-build/targets/bin', trigger=['./build']),
    ]
)

k8s_resource('pets', port_forwards=["8080:8080"],
            extra_pod_selectors=[{'carto.run/workload-name': 'pets-golang','app.kubernetes.io/component': 'run'}])
