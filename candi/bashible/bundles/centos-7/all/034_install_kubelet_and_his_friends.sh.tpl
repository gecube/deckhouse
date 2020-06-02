{{ if eq .kubernetesVersion "1.14" }}
  kubernetes_version="1.14.10-0"
{{ else if eq .kubernetesVersion "1.15" }}
  kubernetes_version="1.15.12-0"
{{ else if eq .kubernetesVersion "1.16" }}
  kubernetes_version="1.16.10-0"
{{ else if eq .kubernetesVersion "1.17" }}
  kubernetes_version="1.17.6-0"
{{ else if eq .kubernetesVersion "1.18" }}
  kubernetes_version="1.18.3-0"
{{ else }}
  {{ fail (printf "Unsupported kubernetes version: %s" .kubernetesVersion) }}
{{ end }}

bb-yum-install "kubelet-$kubernetes_version" "kubectl-$kubernetes_version" kubernetes-cni-0.7.5

if [ ! -f /etc/systemd/system/kubelet.service.d/10-deckhouse.conf ]; then
  systemctl stop kubelet
fi

mkdir -p /etc/kubernetes/manifests
mkdir -p /etc/systemd/system/kubelet.service.d
mkdir -p /etc/kubernetes/pki
mkdir -p /var/lib/kubelet